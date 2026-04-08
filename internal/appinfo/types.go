package appinfo

import "time"

type CachedAppInfo struct {
	FetchedAt time.Time `json:"fetchedAt"`
	App       AppInfo   `json:"app"`
}

type AppInfo struct {
	Name       string                   `json:"name,omitempty"`
	Title      string                   `json:"title,omitempty"`
	AppID      string                   `json:"appId,omitempty"`
	Version    string                   `json:"version,omitempty"`
	Elements   map[string]ElementDefine `json:"elements,omitempty"`
	ExtendApps []AppInfo                `json:"extendApps,omitempty"`
}

type ElementDefine struct {
	FullName       string           `json:"fullName,omitempty"`
	Name           string           `json:"name,omitempty"`
	Title          string           `json:"title,omitempty"`
	Type           string           `json:"type,omitempty"`
	AccessModifier string           `json:"accessModifier,omitempty"`
	FunctionList   []FunctionDef    `json:"functionList,omitempty"`
	FieldList      []map[string]any `json:"fieldList,omitempty"`
	Meta           map[string]any   `json:"meta,omitempty"`
}

type FunctionDef struct {
	Name       string        `json:"name,omitempty"`
	Title      string        `json:"title,omitempty"`
	Args       []FunctionArg `json:"args,omitempty"`
	ReturnType any           `json:"returnType,omitempty"`
}

type FunctionArg struct {
	Name     string `json:"name,omitempty"`
	Title    string `json:"title,omitempty"`
	DataType any    `json:"dataType,omitempty"`
	Generic  any    `json:"generic,omitempty"`
}
