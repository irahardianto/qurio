# Key Learnings: Test Coverage Boost (Jan 2026)

## 1. Backend Testing Patterns (Go)

### Mocking 3rd Party Clients (Weaviate)
**Pattern:** Use `httptest.NewServer` to mock the Weaviate client, but **handle the initialization checks**.
Weaviate client calls `/v1/meta` or `/v1/.well-known/ready` on startup. The mock server handler must verify `r.URL.Path` and return appropriate JSON (e.g., `{"version": "1.19.0"}`) for these checks before asserting on the actual API call.

### Service Layer Testing
**Pattern:** When `Service` is a concrete struct but uses a `Repository` interface, create a `MockRepository` struct in the test package.
This avoids complex refactoring while allowing full isolation of the Service logic.

## 2. Frontend Testing Patterns (Vue/Vitest)

### Store Testing (Pinia)
**Pattern:** Mock `global.fetch` using `vi.fn()` to test actions that make API calls.
Ensure to `mockReset()` in `beforeEach` to avoid state leakage between tests.

### Component Testing (Lucide Icons)
**Pattern:** When testing components with many sub-components (like `lucide-vue-next` icons or UI library components), use **global stubs** to avoid parsing errors and focus on the logic.
```typescript
const globalStubs = {
  Activity: { template: '<svg></svg>' },
  Card: { template: '<div><slot /></div>' }
}
```
