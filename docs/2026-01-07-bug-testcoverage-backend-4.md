1. qurio/apps/backend/main.go
This file acts as the Composition Root, initializing concrete adapters like wstore.NewStore and nsq.NewProducer and passing them to the application.
• Assessment: Integration Test Only.
• The Coverage Gap: Currently sits at 0% coverage because it handles real environment variables, live signal notifications, and enters an infinite blocking loop via application.Run(ctx).
• Improvement Strategy: Do not attempt unit tests here. Instead, implement a "Smoke Test" Integration using Testcontainers.
    ◦ Action: Create a test that executes the main() logic (or a wrapped version) against ephemeral containers for PostgreSQL, Weaviate, and NSQ.
    ◦ Goal: Verify that the application starts, registers routes, and successfully connects to its dependencies without crashing. This will illuminate the entire startup sequence in your coverage report.

2. qurio/apps/backend/internal/app/app.go
This file contains the New constructor which builds the dependency graph and the mux.Handle logic for all HTTP routes and middleware.
• Assessment: Unit Testing (High Priority).
• The Coverage Gap: Coverage is low because while it has been refactored to use Interfaces (Database, VectorStore, TaskPublisher), the current tests may not be exercising all 15+ route registrations or the CORS middleware branches.
• Improvement Strategy: Use the Mocking Strategy already defined in mocks_test.go to achieve 100% logic coverage without a network.
    ◦ Action: Write Table-Driven Unit Tests for the New constructor. Pass MockDatabase (via sqlmock) and MockVectorStore to verify that every route in the mux is correctly registered.
    ◦ Negative Paths: Specifically test if app.New handles failures in sub-services, such as a failure to initialize the FileQueryLogger, which should fall back to os.Stdout.

3. qurio/apps/backend/internal/app/bootstrap.go
This file encapsulates infrastructure initialization, including SQL migrations, DB retry loops, and Weaviate schema checks.
• Assessment: Integration-Heavy Mix.
• The Coverage Gap: This logic is "bolted to the floor" of live infrastructure. Currently, the 10-iteration retry loops for database pings and Weaviate schema checks are "dark" to unit tests.
• Improvement Strategy:
    ◦ Integration Component: Use Testcontainers to run the Bootstrap function against a real starting database. This allows you to verify that the migrate.NewWithDatabaseInstance call correctly applies the SQL files in apps/backend/migrations/*.sql.
    ◦ Unit Component: Mock the VectorStore interface to test the retry logic specifically—simulate 3 failures followed by a success to verify the loop breaks correctly.
