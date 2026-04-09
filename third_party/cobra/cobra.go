package cobra

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type PositionalArgs func(cmd *Command, args []string) error

func ExactArgs(n int) PositionalArgs {
	return func(cmd *Command, args []string) error {
		if len(args) != n {
			return fmt.Errorf("accepts %d arg(s), received %d", n, len(args))
		}
		return nil
	}
}

type flagKind int

const (
	flagString flagKind = iota + 1
	flagBool
	flagInt
)

type flagDef struct {
	name     string
	usage    string
	kind     flagKind
	required bool
	seen     bool
	str      *string
	boolean  *bool
	integer  *int
}

func (f *flagDef) needsValue(inlineValue string) bool {
	return f != nil && f.kind != flagBool && inlineValue == ""
}

type FlagSet struct {
	order []string
	defs  map[string]*flagDef
}

func newFlagSet() *FlagSet {
	return &FlagSet{defs: map[string]*flagDef{}}
}

func (f *FlagSet) StringVar(target *string, name, value, usage string) {
	*target = value
	f.add(&flagDef{name: name, usage: usage, kind: flagString, str: target})
}

func (f *FlagSet) BoolVar(target *bool, name string, value bool, usage string) {
	*target = value
	f.add(&flagDef{name: name, usage: usage, kind: flagBool, boolean: target})
}

func (f *FlagSet) IntVar(target *int, name string, value int, usage string) {
	*target = value
	f.add(&flagDef{name: name, usage: usage, kind: flagInt, integer: target})
}

func (f *FlagSet) add(def *flagDef) {
	if _, exists := f.defs[def.name]; !exists {
		f.order = append(f.order, def.name)
	}
	f.defs[def.name] = def
}

func (f *FlagSet) markRequired(name string) error {
	flag, ok := f.defs[name]
	if !ok {
		return fmt.Errorf("flag %q not defined", name)
	}
	flag.required = true
	return nil
}

type Command struct {
	Use               string
	Short             string
	Long              string
	Example           string
	Version           string
	SilenceErrors     bool
	SilenceUsage      bool
	RunE              func(cmd *Command, args []string) error
	Args              PositionalArgs
	PersistentPreRunE func(cmd *Command, args []string) error

	parent          *Command
	children        []*Command
	flags           *FlagSet
	persistentFlags *FlagSet
	args            []string
	ctx             context.Context
	out             io.Writer
}

func (c *Command) Flags() *FlagSet {
	if c.flags == nil {
		c.flags = newFlagSet()
	}
	return c.flags
}

func (c *Command) PersistentFlags() *FlagSet {
	if c.persistentFlags == nil {
		c.persistentFlags = newFlagSet()
	}
	return c.persistentFlags
}

func (c *Command) AddCommand(children ...*Command) {
	for _, child := range children {
		if child == nil {
			continue
		}
		child.parent = c
		c.children = append(c.children, child)
	}
}

func (c *Command) SetArgs(args []string) {
	c.args = append([]string(nil), args...)
}

func (c *Command) SetOut(w io.Writer) {
	c.out = w
}

func (c *Command) output() io.Writer {
	if c.out != nil {
		return c.out
	}
	if c.parent != nil {
		return c.parent.output()
	}
	return os.Stdout
}

func (c *Command) Context() context.Context {
	if c.ctx == nil {
		return context.Background()
	}
	return c.ctx
}

func (c *Command) Execute() error {
	return c.ExecuteContext(context.Background())
}

func (c *Command) ExecuteContext(ctx context.Context) error {
	args := c.args
	if args == nil {
		args = os.Args[1:]
	}
	if target, ok := c.helpTarget(args); ok {
		target.ctx = ctx
		target.printHelp()
		return nil
	}

	leaf, positionals, chain, err := c.resolve(ctx, args, nil, nil)
	if err != nil {
		return err
	}
	for _, item := range chain {
		item.ctx = ctx
	}
	for _, item := range chain {
		if item.PersistentPreRunE != nil {
			if err := item.PersistentPreRunE(leaf, positionals); err != nil {
				return err
			}
		}
	}
	if leaf.Args != nil {
		if err := leaf.Args(leaf, positionals); err != nil {
			return err
		}
	}
	for _, item := range chain {
		if err := ensureRequired(item.PersistentFlags()); err != nil {
			return err
		}
	}
	if err := ensureRequired(leaf.Flags()); err != nil {
		return err
	}
	if leaf.RunE != nil {
		return leaf.RunE(leaf, positionals)
	}
	leaf.printHelp()
	return nil
}

func (c *Command) MarkFlagRequired(name string) error {
	if c.flags != nil {
		if err := c.flags.markRequired(name); err == nil {
			return nil
		}
	}
	if c.persistentFlags != nil {
		if err := c.persistentFlags.markRequired(name); err == nil {
			return nil
		}
	}
	return fmt.Errorf("flag %q not defined", name)
}

func (c *Command) resolve(ctx context.Context, args []string, inherited []*FlagSet, chain []*Command) (*Command, []string, []*Command, error) {
	c.ctx = ctx
	chain = append(chain, c)
	available := append([]*FlagSet{}, inherited...)
	if c.persistentFlags != nil {
		available = append(available, c.persistentFlags)
	}
	if c.flags != nil {
		available = append(available, c.flags)
	}

	child, childIndex, err := c.findChild(args, available)
	if err != nil {
		return nil, nil, nil, err
	}
	if child != nil {
		if _, err := parseArgs(args[:childIndex], available); err != nil {
			return nil, nil, nil, err
		}
		nextInherited := append([]*FlagSet{}, inherited...)
		if c.persistentFlags != nil {
			nextInherited = append(nextInherited, c.persistentFlags)
		}
		return child.resolve(ctx, args[childIndex+1:], nextInherited, chain)
	}

	positionals, err := parseArgs(args, available)
	if err != nil {
		return nil, nil, nil, err
	}
	return c, positionals, chain, nil
}

func (c *Command) findChild(args []string, available []*FlagSet) (*Command, int, error) {
	flagMap := mergeFlagSets(available)
	for i := 0; i < len(args); i++ {
		token := args[i]
		if strings.HasPrefix(token, "-") {
			name, inlineValue := splitFlagToken(token)
			if isHelpFlag(name) {
				return nil, -1, nil
			}
			def := flagMap[name]
			if def == nil {
				return nil, -1, fmt.Errorf("unknown flag: --%s", name)
			}
			if def.needsValue(inlineValue) {
				i++
				if i >= len(args) {
					return nil, -1, fmt.Errorf("flag needs an argument: --%s", name)
				}
			}
			continue
		}
		if token == "help" {
			return nil, -1, nil
		}
		for _, child := range c.children {
			if child.name() == token {
				return child, i, nil
			}
		}
	}
	return nil, -1, nil
}

func (c *Command) helpTarget(args []string) (*Command, bool) {
	if len(args) == 0 {
		return nil, false
	}
	if len(args) >= 1 && args[0] == "help" {
		return c.followCommandPath(args[1:]), true
	}

	current := c
	for i := 0; i < len(args); i++ {
		token := args[i]
		if token == "--help" || token == "-h" {
			return current, true
		}
		if strings.HasPrefix(token, "-") {
			name, inlineValue := splitFlagToken(token)
			if isHelpFlag(name) {
				return current, true
			}
			flagMap := mergeFlagSets(current.flagLineage())
			if def := flagMap[name]; def.needsValue(inlineValue) {
				i++
			}
			continue
		}
		child := current.lookupChild(token)
		if child == nil {
			continue
		}
		current = child
	}
	return nil, false
}

func (c *Command) followCommandPath(tokens []string) *Command {
	current := c
	for _, token := range tokens {
		if strings.HasPrefix(token, "-") {
			break
		}
		child := current.lookupChild(token)
		if child == nil {
			break
		}
		current = child
	}
	return current
}

func (c *Command) lookupChild(name string) *Command {
	for _, child := range c.children {
		if child.name() == name {
			return child
		}
	}
	return nil
}

func (c *Command) flagLineage() []*FlagSet {
	var sets []*FlagSet
	for current := c; current != nil; current = current.parent {
		if current.persistentFlags != nil {
			sets = append([]*FlagSet{current.persistentFlags}, sets...)
		}
	}
	return sets
}

func (c *Command) name() string {
	fields := strings.Fields(strings.TrimSpace(c.Use))
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func (c *Command) commandPath() string {
	if c.parent == nil || c.parent.name() == "" {
		return c.name()
	}
	parentPath := c.parent.commandPath()
	if parentPath == "" {
		return c.name()
	}
	return parentPath + " " + c.name()
}

func (c *Command) usageLine() string {
	use := strings.TrimSpace(c.Use)
	if c.parent == nil {
		return use
	}
	parentPath := c.parent.commandPath()
	if parentPath == "" {
		return use
	}
	return parentPath + " " + use
}

func (c *Command) printHelp() {
	out := c.output()
	short := strings.TrimSpace(c.Short)
	long := strings.TrimSpace(c.Long)
	example := strings.TrimSpace(c.Example)
	if short != "" {
		_, _ = fmt.Fprintln(out, short)
		_, _ = fmt.Fprintln(out)
	}
	if long != "" {
		_, _ = fmt.Fprintln(out, long)
		_, _ = fmt.Fprintln(out)
	}
	_, _ = fmt.Fprintln(out, "用法:")
	_, _ = fmt.Fprintf(out, "  %s\n", c.usageLine())

	if len(c.children) > 0 {
		_, _ = fmt.Fprintln(out)
		_, _ = fmt.Fprintln(out, "可用命令:")
		for _, child := range c.children {
			_, _ = fmt.Fprintf(out, "  %-12s %s\n", child.name(), child.Short)
		}
	}

	if flags := c.PersistentFlags(); flags != nil && len(flags.defs) > 0 {
		c.printFlags(out, "全局参数:", flags)
	}
	if flags := c.Flags(); flags != nil && len(flags.defs) > 0 {
		c.printFlags(out, "参数:", flags)
	}
	if example != "" {
		_, _ = fmt.Fprintln(out)
		_, _ = fmt.Fprintln(out, "示例:")
		_, _ = fmt.Fprintln(out, example)
	}
}

func (c *Command) printFlags(out io.Writer, title string, flags *FlagSet) {
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, title)
	for _, name := range flags.order {
		def := flags.defs[name]
		if def == nil {
			continue
		}
		line := fmt.Sprintf("  --%-12s %s", def.name, def.usage)
		if def.required {
			line += "（必填）"
		}
		_, _ = fmt.Fprintln(out, line)
	}
}

func parseArgs(args []string, sets []*FlagSet) ([]string, error) {
	flagMap := mergeFlagSets(sets)
	positionals := make([]string, 0)

	for i := 0; i < len(args); i++ {
		token := args[i]
		if !strings.HasPrefix(token, "-") {
			positionals = append(positionals, token)
			continue
		}

		name, inlineValue := splitFlagToken(token)
		if isHelpFlag(name) {
			continue
		}
		def := flagMap[name]
		if def == nil {
			return nil, fmt.Errorf("unknown flag: --%s", name)
		}

		value := inlineValue
		if def.kind == flagBool {
			if value == "" {
				value = "true"
			}
		} else if def.needsValue(value) {
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("flag needs an argument: --%s", name)
			}
			value = args[i]
		}

		if err := applyFlagValue(def, value); err != nil {
			return nil, err
		}
		def.seen = true
	}
	return positionals, nil
}

func mergeFlagSets(sets []*FlagSet) map[string]*flagDef {
	result := map[string]*flagDef{}
	for _, set := range sets {
		if set == nil {
			continue
		}
		for name, def := range set.defs {
			result[name] = def
		}
	}
	return result
}

func splitFlagToken(token string) (string, string) {
	clean := strings.TrimPrefix(strings.TrimPrefix(token, "--"), "-")
	if idx := strings.Index(clean, "="); idx >= 0 {
		return clean[:idx], clean[idx+1:]
	}
	return clean, ""
}

func isHelpFlag(name string) bool {
	return name == "help" || name == "h"
}

func applyFlagValue(def *flagDef, value string) error {
	switch def.kind {
	case flagString:
		*def.str = value
	case flagBool:
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean for --%s: %w", def.name, err)
		}
		*def.boolean = parsed
	case flagInt:
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid integer for --%s: %w", def.name, err)
		}
		*def.integer = parsed
	default:
		return fmt.Errorf("unsupported flag kind for --%s", def.name)
	}
	return nil
}

func ensureRequired(flags *FlagSet) error {
	if flags == nil {
		return nil
	}
	for _, def := range flags.defs {
		if def.required && !def.seen {
			return fmt.Errorf("必填参数 %q 未设置", "--"+def.name)
		}
	}
	return nil
}
