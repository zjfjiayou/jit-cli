#!/usr/bin/env node
"use strict";

const fs = require("fs");
const os = require("os");
const path = require("path");
const cp = require("child_process");

const pkg = require("../package.json");
const version = pkg.version;
const repo = process.env.JIT_CLI_REPO || "wanyun/JitCli";
const binName = process.env.JIT_CLI_BIN_NAME || "jit";

const platformMap = {
  darwin: "darwin",
  linux: "linux",
  win32: "windows",
};

const archMap = {
  x64: "amd64",
  arm64: "arm64",
};

const platform = platformMap[process.platform];
const arch = archMap[process.arch];

if (!platform || !arch) {
  console.error(`Unsupported platform: ${process.platform}/${process.arch}`);
  process.exit(1);
}

const isWindows = process.platform === "win32";
const ext = isWindows ? ".zip" : ".tar.gz";
const archiveName = `${binName}-${platform}-${arch}${ext}`;
const url = `https://github.com/${repo}/releases/download/v${version}/${archiveName}`;

const root = path.join(__dirname, "..");
const vendorDir = path.join(root, "scripts", "vendor");
const outBinary = path.join(vendorDir, isWindows ? `${binName}.exe` : binName);

function ensureCleanDir(dir) {
  fs.rmSync(dir, { recursive: true, force: true });
  fs.mkdirSync(dir, { recursive: true });
}

function download(dest) {
  cp.execFileSync(
    "curl",
    ["--fail", "--location", "--silent", "--show-error", "--output", dest, url],
    { stdio: "inherit" }
  );
}

function extract(archivePath, extractDir) {
  if (isWindows) {
    cp.execFileSync(
      "powershell.exe",
      [
        "-NoLogo",
        "-NoProfile",
        "-Command",
        `Expand-Archive -Path '${archivePath.replace(/'/g, "''")}' -DestinationPath '${extractDir.replace(/'/g, "''")}' -Force`,
      ],
      { stdio: "inherit" }
    );
    return;
  }
  cp.execFileSync("tar", ["-xzf", archivePath, "-C", extractDir], { stdio: "inherit" });
}

function findBinary(rootDir) {
  const queue = [rootDir];
  while (queue.length > 0) {
    const current = queue.shift();
    const entries = fs.readdirSync(current, { withFileTypes: true });
    for (const entry of entries) {
      const p = path.join(current, entry.name);
      if (entry.isDirectory()) {
        queue.push(p);
      } else if (entry.name === binName || entry.name === `${binName}.exe`) {
        return p;
      }
    }
  }
  return "";
}

function main() {
  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "jit-cli-npm-"));
  const archivePath = path.join(tempDir, archiveName);
  const extractDir = path.join(tempDir, "extract");
  fs.mkdirSync(extractDir, { recursive: true });

  try {
    download(archivePath);
    extract(archivePath, extractDir);
    const binary = findBinary(extractDir);
    if (!binary) {
      throw new Error(`binary ${binName} not found in archive`);
    }
    ensureCleanDir(vendorDir);
    fs.copyFileSync(binary, outBinary);
    if (!isWindows) {
      fs.chmodSync(outBinary, 0o755);
    }
    console.log(`${binName} ${version} installed to npm wrapper`);
  } catch (err) {
    console.error(`Failed to install ${binName}: ${err.message}`);
    process.exit(1);
  } finally {
    fs.rmSync(tempDir, { recursive: true, force: true });
  }
}

main();
