import sys
import os
from unittest.mock import MagicMock

# Add project root to sys.path so that tests can import modules like config, handlers, logger
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..")))

# Mock crawl4ai modules to avoid installation issues
# We need to mock them BEFORE any test imports handlers.web
module_names = [
    "crawl4ai",
    "crawl4ai.content_filter_strategy",
    "crawl4ai.markdown_generation_strategy",
    "docling",
    "docling.document_converter",
    "docling.datamodel",
    "docling.datamodel.pipeline_options",
    "docling.datamodel.base_models",
]

for name in module_names:
    if name not in sys.modules:
        sys.modules[name] = MagicMock()
