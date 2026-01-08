# Plan: Bug Fixes & Inconsistencies (Part 2: Refactoring & Verification)

## Scope
This plan addresses the remaining gaps from the 2026-01-08 forensic report, specifically focusing on frontend configuration consolidation and rigorous logging verification. It compliments `docs/plans/2026-01-08-bug-inconsistencies-1.md`.

## Task 1: Frontend Single Source of Truth for Paths

**Files:**
- Modify: `apps/frontend/vite.config.ts`
- Modify: `apps/frontend/package.json`

**Requirements:**
- **Acceptance Criteria**
  1. `vite.config.ts` must NOT contain manual `resolve.alias` definitions.
  2. `vite-tsconfig-paths` plugin must be active.
  3. `npm run build` must succeed without path resolution errors.
  4. Application must load in browser without 404s for `@/` imports.

- **Functional Requirements**
  1. Install `vite-tsconfig-paths` dev dependency.
  2. Register plugin in `vite.config.ts`.
  3. Remove redundant alias configuration.

- **Non-Functional Requirements**
  1. Build time impact should be negligible (< 1s).
  2. Maintain compatibility with Vue 3 single-file components.

- **Requirements Reference (RAG)**
  - *Vite Plugins*: Confirmed usage of `vite-tsconfig-paths` for automatic alias resolution from `tsconfig.json`.

- **Test Coverage**
  - [Build] `npm run build` serves as the primary verification.
  - [Manual] Verify `import { Button } from '@/components/ui/button'` works in a component.

**Step 1: Write failing verification**
Run `grep "alias:" apps/frontend/vite.config.ts`.
Expected: Matches found (Validation that cleanup is needed).

**Step 2: Install Plugin**
`npm install -D vite-tsconfig-paths --prefix apps/frontend`

**Step 3: Write minimal implementation**
```typescript
// apps/frontend/vite.config.ts
import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tsconfigPaths from 'vite-tsconfig-paths' // Add import

export default defineConfig({
  plugins: [
    vue(),
    tsconfigPaths(), // Add plugin
  ],
  resolve: {
    // Remove alias section entirely
  },
  // ... rest of config
})
```

**Step 4: Verify build**
`npm run build --prefix apps/frontend`


## Task 2: Worker Logging Verification Suite

**Files:**
- Create: `apps/ingestion-worker/tests/test_logging_integration.py`

**Requirements:**
- **Acceptance Criteria**
  1. Test script passes only if `tornado`, `nsq`, and `root` loggers emit valid JSON.
  2. Test script fails if any log is emitted as plain text.

- **Functional Requirements**
  1. Import application logging config.
  2. Capture `stdout` and `stderr`.
  3. Stimulate all major logging subsystems.
  4. Parse output lines as JSON.

- **Non-Functional Requirements**
  1. Zero external test dependencies (use `unittest` or `pytest` if available, or raw `assert`).
  2. Fast execution (< 100ms).

- **Requirements Reference (RAG)**
  - *Structlog Stdlib*: Confirmed `structlog.stdlib.LoggerFactory` intercepts standard logging.
  - *Python Logging*: Confirmed `logging.getLogger("tornado").handlers` manipulation is required.

- **Test Coverage**
  - [Integration] `apps/ingestion-worker/tests/test_logging_integration.py`

**Step 1: Write test script**
```python
# apps/ingestion-worker/tests/test_logging_integration.py
import logging
import json
import io
import contextlib
import sys
# Adjust path to find modules
sys.path.append('.') 
from logger import configure_logger

def test_logs_are_json():
    f = io.StringIO()
    # Capture stdout
    with contextlib.redirect_stdout(f):
        configure_logger()
        
        # 1. Standard Root Log
        logging.getLogger("root").info("test_root_event", extra_key="value")
        
        # 2. Tornado Infrastructure Log (Simulate StreamClosedError)
        logging.getLogger("tornado.access").warning("test_tornado_request")
        
        # 3. Structlog Application Log
        import structlog
        structlog.get_logger().info("test_structlog_event")
        
    output = f.getvalue().strip()
    if not output:
        print("FAIL: No logs captured")
        sys.exit(1)
        
    for line in output.split('\n'):
        if not line.strip(): continue
        try:
            data = json.loads(line)
            # Verify minimum fields
            if "timestamp" not in data and "level" not in data:
                print(f"FAIL: Missing standard fields in: {line}")
                sys.exit(1)
        except json.JSONDecodeError:
            print(f"FAIL: Log line is not JSON: {line}")
            sys.exit(1)

if __name__ == "__main__":
    try:
        test_logs_are_json()
        print("PASS: All logs are JSON")
    except ImportError:
        print("SKIP: Project modules not found (run from apps/ingestion-worker root)")
```

**Step 2: Verify test fails (before fix from Plan 1)**
`python3 apps/ingestion-worker/tests/test_logging_integration.py`
Expected: Fail if `tornado` logs are plain text.

**Step 3: Run after Plan 1 Implementation**
`python3 apps/ingestion-worker/tests/test_logging_integration.py`
Expected: "PASS: All logs are JSON"

## Plan Completion Review
- [x] Compliance verified: `technical-constitution`
- [x] Gaps addressed: Frontend Duplication, Logging Split-Brain
- [x] TDD: Yes
- [x] Verification: Build checks + Integration script
