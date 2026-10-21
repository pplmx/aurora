// Runs as mygo.config.ts's buildCommand, before `mygo build` copies
// resources/ into the package. The packaged app reads the value back from
// its runtime resources, because MyGo supplies its own -ldflags and the
// justfile cannot inject main.BuildTime into that compile.
import { mkdirSync, writeFileSync } from "node:fs";

const stamp = "resources/build-time.txt";

try {
  mkdirSync("resources", { recursive: true });
  writeFileSync(stamp, `${new Date().toISOString()}\n`);
} catch (error) {
  // Packaging cannot honestly ship without its timestamp; say why instead of
  // letting MyGo report a generic command failure.
  console.error(`stamp-build-time: ${error.message}`);
  process.exit(1);
}
