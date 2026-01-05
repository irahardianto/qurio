# Application Enhancement Proposal: Granular Control for Qurio Documentation Tools

**Date:** January 5, 2026
**Author:** AI Agent (Gemini)
**Status:** Draft
**Target Component:** Qurio Tool Suite (MCP)

## 1. Executive Summary
This proposal outlines specific enhancements to the Qurio documentation retrieval tools (`qurio_list_pages` and `qurio_read_page`). The objective is to optimize the interaction efficiency for AI agents by implementing granular filtering and content retrieval mechanisms. These changes will significantly reduce token consumption, improve response latency, and minimize context window pollution.

## 2. Problem Statement
The current implementation of the Qurio toolset operates on a "all-or-nothing" basis for page listing and content retrieval:
*   **Discovery Inefficiency:** `qurio_list_pages` returns the entire site map for a source. For large documentation sets (e.g., AWS docs, MDN), this JSON payload can be massive, consuming valuable tokens just to find a single relevant URL.
*   **Retrieval Inefficiency:** `qurio_read_page` retrieves the full content of a document. When an agent only needs a specific section (e.g., "Mocking Getters" within a large "Testing" guide), loading the entire document is wasteful and distracts the model with irrelevant context.

## 3. Proposed Enhancements

### 3.1 Feature: Server-Side Page Filtering
**Target Tool:** `qurio_list_pages`

**Requirement:**
Modify the `qurio_list_pages` tool to accept an optional filtering argument.

**Technical Specification:**
*   **New Parameter:** `filter` (string, optional).
*   **Behavior:**
    *   If `filter` is provided, the backend should perform a case-insensitive substring match against the page `url` and `title` (if available).
    *   Only matching entries are returned in the JSON array.
*   **Example Call:** `qurio_list_pages(source_id="...", filter="testing")`

**Benefit:**
Reduces the output size from hundreds of lines to just the few relevant entries, allowing the agent to quickly identify the correct target without processing the entire site structure.

### 3.2 Feature: Section-Level Content Retrieval
**Target Tool:** `qurio_read_page`

**Requirement:**
Modify the content retrieval tool to allow fetching specific sections of a document rather than the whole file.

**Technical Specification:**
*   **New Parameter:** `section` (string, optional) OR `selector` (string, optional).
*   **Behavior:**
    *   **Markdown Sources:** If `section` matches a header (e.g., `## Introduction`), return content only until the next header of the same level.
    *   **HTML/Web Sources:** If `selector` (e.g., CSS selector or anchor ID) is provided, return the text content of that element and its immediate children.
    *   **Fallback:** If the section is not found, return an informative error or the full page (configurable behavior).
*   **Example Call:** `qurio_read_page(url="...", section="#mocking-getters")`

**Benefit:**
Drastically reduces the token count for retrieval operations. It allows the agent to perform "surgical" reads, extracting only the specific answers needed.

## 4. Impact Analysis

| Metric | Current State | Projected State |
| :--- | :--- | :--- |
| **Token Usage (Discovery)** | High (Full list) | Low (Filtered list) |
| **Token Usage (Reading)** | High (Full page) | Low/Medium (Targeted section) |
| **Latency** | Higher (Large payloads) | Lower (Small payloads) |
| **Agent Accuracy** | Risk of distraction from noise | Higher focus on relevant text |

## 5. Recommendation
We recommend prioritizing **Feature 3.1 (Page Filtering)** as it involves lower technical complexity while offering immediate token savings for large documentation sources. **Feature 3.2 (Section Retrieval)** should follow, potentially requiring more sophisticated parsing logic on the backend.
