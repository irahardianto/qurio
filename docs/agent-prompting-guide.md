# Agent Prompting Guide

This guide helps you effectively use Qurio's MCP tools with your AI coding assistant (Claude, Gemini CLI, etc.).

## Core Philosophy
Qurio acts as a **Shared Library** for your agent. Instead of relying on the agent's outdated training data, you should encourage it to "consult the library" (Qurio) to get the ground truth.

## The Toolset

### `qurio_search`
**Goal:** Find specific information.
**Key Args:** `query` (required).
**Best Practice:** Encourage specific queries. "How to auth with Clerk" is better than "Auth".

### `qurio_list_sources`
**Goal:** Discover what is known.
**Best Practice:** Use this when starting a task to verify if the necessary context is available.

### `qurio_list_pages`
**Goal:** Explore a specific topic deeply.
**Best Practice:** Use after listing sources to drill down into a specific library's documentation structure.

### `qurio_read_page`
**Goal:** Get the full context.
**Best Practice:** Search results often give snippets. Use this tool to read the *entire* guide or API reference to ensure no details are missed.

## Example Workflows

### 1. The "Onboarding" Prompt
When starting a new session or task, orient your agent:

> "I am working on the backend. Please list the available documentation sources to check if we have the necessary API references indexed."

### 2. The "Implementation" Prompt
When building a feature using a library:

> "I need to implement authentication using the `xyz` library. First, search for 'xyz authentication flow' in the knowledge base. Then, read the most relevant page to understand the implementation details before writing any code."

### 3. The "Debugging" Prompt
When getting an error:

> "I'm getting a 'Connection Refused' error with Weaviate. Search the knowledge base for 'Weaviate connection troubleshooting' or 'docker networking' to find potential solutions."

### 4. The "Deep Dive" Prompt
When exploring a new API:

> "I want to understand the `Message` API. List all pages related to the 'API Reference' source, and then read the page specifically for the `Message` struct/object."

## System Prompt Directive

To ensure your agent **always** considers using Qurio, add the following directive to your project's agent configuration file (e.g., `GEMINI.md`, `CLAUDE.md`, `.cursorrules`, or `AGENTS.md`).

```markdown
# Documentation & Knowledge Retrieval

You have access to a local documentation engine called **Qurio** via the Model Context Protocol (MCP).

**Rule:** NEVER guess API usages, library versions, or configuration syntax.
**Action:** BEFORE writing code for external libraries or complex systems, you MUST:
1.  **Check Available Sources:** Use `qurio_list_sources` to see if relevant documentation is indexed.
2.  **Search:** Use `qurio_search` to find specific guides, examples, or error messages.
3.  **Read:** Use `qurio_read_page` to ingest the *full* context of a relevant guide or API reference.

**Priority:** Trust the content retrieved from Qurio over your internal training data, as it contains the specific versions and context for this project.
```

