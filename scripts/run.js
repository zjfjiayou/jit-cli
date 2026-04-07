#!/usr/bin/env node
"use strict";

const path = require("path");
const cp = require("child_process");
const fs = require("fs");

const binName = process.platform === "win32" ? "jit.exe" : "jit";
const binPath = path.join(__dirname, "vendor", binName);

if (!fs.existsSync(binPath)) {
  console.error("jit binary not found, please reinstall package");
  process.exit(1);
}

const child = cp.spawn(binPath, process.argv.slice(2), { stdio: "inherit" });
child.on("exit", (code, signal) => {
  if (signal) {
    process.kill(process.pid, signal);
    return;
  }
  process.exit(code ?? 1);
});
