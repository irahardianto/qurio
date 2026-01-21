# 2026-01-21-ingestion-retry-config-1

## Step 1: Extract Requirements

**✓ Requirements Extracted**
- **Scope**: Expose ingestion worker retry settings (`retry_max_attempts`, `retry_initial_delay_ms`, `retry_max_delay_ms`, `retry_backoff_multiplier`) to the root `.env` file and propagate them via `docker-compose.yml`.
- **Gap Analysis**:
    - Nouns: `.env.example`, `docker-compose.yml`, `ingestion-worker` services, retry configuration variables.
    - Verbs: Add variables, Propagate to containers, Verify overrides.
- **Exclusions**:
    - Modifying `apps/ingestion-worker/config.py` logic (defaults will remain as code-level fallbacks).

## Step 2: Knowledge Enrichment

**✓ Knowledge Enrichment**
- **RAG Queries**:
    - `pydantic-settings` env var precedence -> Confirmed Env Vars > Defaults.
    - `docker-compose` environment syntax -> Confirmed `- VAR=${VAR:-default}` pattern.
- **Analysis**:
    - `apps/ingestion-worker/config.py` uses `pydantic-settings`.
    - `docker-compose.yml` defines `ingestion-worker-web` and `ingestion-worker-file` services which need these variables.

## Step 3: Generate Implementation Plan

### Task 1: Update .env.example

**Files:**
- Modify: `.env.example`

**Requirements:**
- **Acceptance Criteria**
  1. `.env.example` includes `RETRY_MAX_ATTEMPTS`, `RETRY_INITIAL_DELAY_MS`, `RETRY_MAX_DELAY_MS`, `RETRY_BACKOFF_MULTIPLIER`.
  2. Defaults match current code values (3, 1000, 60000, 2).
  3. Comments explain units (ms) and purpose.

- **Functional Requirements**
  1. Provide a central template for configuring retry behavior.

- **Non-Functional Requirements**
  None for this task.

- **Test Coverage**
  - Manual verification that file exists and contains new lines.

**Step 1: Write failing test**
*Skipped: Documentation/Config template change only.*

**Step 2: Verify test fails**
*Skipped*

**Step 3: Write minimal implementation**
```bash
# Append to .env.example
cat <<EOT >> .env.example

# Ingestion Worker Retry Settings
RETRY_MAX_ATTEMPTS=3
RETRY_INITIAL_DELAY_MS=1000
RETRY_MAX_DELAY_MS=60000
RETRY_BACKOFF_MULTIPLIER=2
EOT
```

**Step 4: Verify test passes**
```bash
grep "RETRY_MAX_ATTEMPTS" .env.example
```

### Task 2: Propagate via Docker Compose

**Files:**
- Modify: `docker-compose.yml`

**Requirements:**
- **Acceptance Criteria**
  1. `ingestion-worker-web` service receives the 4 retry variables in its `environment` section.
  2. `ingestion-worker-file` service receives the 4 retry variables in its `environment` section.
  3. Use `${VAR:-default}` syntax to ensure backward compatibility if `.env` is missing them (though `.env.example` updates suggest they should be set).

- **Functional Requirements**
  1. Enable runtime configuration of retries without rebuilding images.

- **Non-Functional Requirements**
  None for this task.

- **Test Coverage**
  - `docker compose config` validates syntax and variable substitution.

**Step 1: Write failing test**
*Skipped: Configuration file change.*

**Step 2: Verify test fails**
*Skipped*

**Step 3: Write minimal implementation**
```yaml
# Add to environment section of ingestion-worker-web AND ingestion-worker-file in docker-compose.yml
    environment:
      # ... existing vars ...
      - RETRY_MAX_ATTEMPTS=${RETRY_MAX_ATTEMPTS:-3}
      - RETRY_INITIAL_DELAY_MS=${RETRY_INITIAL_DELAY_MS:-1000}
      - RETRY_MAX_DELAY_MS=${RETRY_MAX_DELAY_MS:-60000}
      - RETRY_BACKOFF_MULTIPLIER=${RETRY_BACKOFF_MULTIPLIER:-2}
```

**Step 4: Verify test passes**
```bash
docker compose config | grep "RETRY_MAX_ATTEMPTS"
```

### Task 3: Integration Verification

**Files:**
- Create: `apps/ingestion-worker/tests/test_env_config.py`

**Requirements:**
- **Acceptance Criteria**
  1. Test verifies that `Settings` loads values from environment variables when present.
  2. Test verifies that `Settings` falls back to defaults when environment variables are missing.
  3. Use `monkeypatch` fixture for safe environment manipulation.

- **Functional Requirements**
  1. Confirm `pydantic-settings` behavior matches expectations.

- **Non-Functional Requirements**
  1. Fast execution (<1s).

- **Test Coverage**
  - [Unit] `test_settings_load_from_env`
  - [Unit] `test_settings_defaults`

**Step 1: Write failing test**
```python
# apps/ingestion-worker/tests/test_env_config.py
from config import Settings
import pytest

def test_settings_load_from_env(monkeypatch):
    # Arrange
    monkeypatch.setenv("RETRY_MAX_ATTEMPTS", "10")
    monkeypatch.setenv("RETRY_INITIAL_DELAY_MS", "500")
    monkeypatch.setenv("RETRY_MAX_DELAY_MS", "30000")
    monkeypatch.setenv("RETRY_BACKOFF_MULTIPLIER", "3")

    # Act
    settings = Settings()

    # Assert
    assert settings.retry_max_attempts == 10
    assert settings.retry_initial_delay_ms == 500
    assert settings.retry_max_delay_ms == 30000
    assert settings.retry_backoff_multiplier == 3

def test_settings_defaults(monkeypatch):
    # Arrange: Ensure strict isolation by removing vars if they exist
    monkeypatch.delenv("RETRY_MAX_ATTEMPTS", raising=False)
    monkeypatch.delenv("RETRY_INITIAL_DELAY_MS", raising=False)
    monkeypatch.delenv("RETRY_MAX_DELAY_MS", raising=False)
    monkeypatch.delenv("RETRY_BACKOFF_MULTIPLIER", raising=False)

    # Act
    settings = Settings()

    # Assert: Should match defaults in config.py
    assert settings.retry_max_attempts == 3
    assert settings.retry_initial_delay_ms == 1000
    assert settings.retry_max_delay_ms == 60000
    assert settings.retry_backoff_multiplier == 2
```

**Step 2: Verify test fails**
*Note: Run pytest. Expect success if Pydantic defaults are standard case-insensitive.*
Run: `pytest apps/ingestion-worker/tests/test_env_config.py`

**Step 3: Write minimal implementation**
*None expected unless config.py requires `case_sensitive=True` adjustments.*

**Step 4: Verify test passes**
Run: `pytest apps/ingestion-worker/tests/test_env_config.py`

## Step 4: Plan Completion Review

**✓ Plan Review Complete**
- Compliance verified: `technical-constitution` (Configuration Management: Separation of Config and Code).
- Testing Strategy: Updated to use `pytest` fixtures for isolation.
- Serena memories updated: No critical updates needed yet.