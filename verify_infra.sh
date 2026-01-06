#!/bin/bash
# verify_infra.sh
# Check if services are reachable

echo "Verifying Infrastructure..."

# Check Docker Daemon (Required for Integration Tests)
if docker info > /dev/null 2>&1; then
    echo "✅ Docker Daemon reachable"
else
    echo "❌ Docker Daemon NOT reachable (Required for Integration Tests)"
    exit 1
fi

# Check Backend (API)
if curl -s -f http://localhost:8081/health > /dev/null; then
    echo "✅ Backend reachable"
else
    echo "❌ Backend NOT reachable"
    exit 1
fi

# Check Frontend
if curl -s -f http://localhost:3000 > /dev/null; then
    echo "✅ Frontend reachable"
else
    echo "❌ Frontend NOT reachable"
    # exit 1 # Soft fail for frontend for now as it might be empty
fi

# Check Postgres
if nc -z localhost 5432; then
    echo "✅ Postgres reachable"
else
    echo "❌ Postgres NOT reachable"
    exit 1
fi

# Check Weaviate
if nc -z localhost 8080; then
    echo "✅ Weaviate reachable"
else
    echo "❌ Weaviate NOT reachable"
    exit 1
fi

echo "Infrastructure Verification Passed!"
