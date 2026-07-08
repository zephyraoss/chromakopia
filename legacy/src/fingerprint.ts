import { spawn } from "bun";
import { writeFileSync, unlinkSync } from "fs";

export interface FpcalcResult {
  duration: number;
  fingerprint: string;
  file: string;
}

export async function runFpcalc(filePath: string): Promise<FpcalcResult> {
  const proc = spawn(["fpcalc", filePath]);

  const stderr = proc.stderr ? await proc.stderr.text() : "";
  const output = await proc.stdout.text();
  
  const exitCode = await proc.exited;

  if (exitCode !== 0 || output.toLowerCase().includes("error:")) {
    const errorMsg = output.trim() || stderr || "unknown error";
    throw new Error(`fpcalc failed: ${errorMsg}`);
  }

  const result: FpcalcResult = {
    duration: 0,
    fingerprint: "",
    file: "",
  };

  for (const line of output.trim().split("\n")) {
    const [key, ...valueParts] = line.split("=");
    const value = valueParts.join("=");
    if (key === "DURATION") {
      result.duration = Math.round(parseFloat(value));
    } else if (key === "FINGERPRINT") {
      result.fingerprint = value;
    } else if (key === "FILE") {
      result.file = value;
    }
  }

  if (!result.fingerprint) {
    throw new Error("No fingerprint generated");
  }

  return result;
}

export async function runFpcalcOnUpload(
  file: File,
  filename: string
): Promise<FpcalcResult> {
  const tempPath = `/tmp/${Date.now()}-${filename}`;
  const buffer = await file.arrayBuffer();
  writeFileSync(tempPath, Buffer.from(buffer));

  try {
    return await runFpcalc(tempPath);
  } finally {
    try { unlinkSync(tempPath); } catch {}
  }
}

export async function runFpcalcOnUrl(url: string): Promise<FpcalcResult> {
  const tempPath = `/tmp/${Date.now()}-${url.split("/").pop() || "audio"}`;

  try {
    const response = await fetch(url);
    if (!response.ok) {
      throw new Error(`Failed to download: ${response.status} ${response.statusText}`);
    }
    const buffer = await response.arrayBuffer();
    writeFileSync(tempPath, Buffer.from(buffer));

    return await runFpcalc(tempPath);
  } finally {
    try { unlinkSync(tempPath); } catch {}
  }
}
