import pytest
from unittest.mock import MagicMock
from handlers.file import extract_metadata_from_doc
from handlers.web import extract_web_metadata

# --- File Metadata Tests ---


@pytest.mark.parametrize(
    "doc_mock, result_mock, file_path, expected_meta",
    [
        # Case 1: Metadata present in doc object
        (
            MagicMock(
                metadata=MagicMock(
                    title="Doc Title",
                    authors=["Alice"],
                    creation_date="2023-01-01",
                    language="fr",
                ),
                num_pages=5,
            ),
            MagicMock(),
            "/tmp/test.pdf",
            {
                "title": "Doc Title",
                "author": "Alice",
                "created_at": "2023-01-01",
                "pages": 5,
                "language": "fr",
            },
        ),
        # Case 2: Metadata missing, fallback to filename and result pages
        (
            MagicMock(
                metadata=None, origin=MagicMock(filename="origin.pdf"), num_pages=None
            ),
            MagicMock(pages=[1, 2, 3]),
            "/tmp/fallback.pdf",
            {
                "title": "origin.pdf",
                "author": None,
                "created_at": None,
                "pages": 3,
                "language": "en",
            },
        ),
        # Case 3: Metadata missing, fallback to basename
        (
            MagicMock(metadata=None, origin=None, num_pages=None),
            MagicMock(pages=[]),
            "/path/to/base.pdf",
            {
                "title": "base.pdf",
                "author": None,
                "created_at": None,
                "pages": 0,
                "language": "en",
            },
        ),
        # Case 4: Authors is a list
        (
            MagicMock(
                metadata=MagicMock(
                    title="T",
                    authors=["Alice", "Bob"],
                    creation_date=None,
                    language="en",
                ),
                num_pages=1,
            ),
            MagicMock(),
            "f.pdf",
            {
                "title": "T",
                "author": "Alice, Bob",
                "created_at": None,
                "pages": 1,
                "language": "en",
            },
        ),
        # Case 5: Callable values (pydantic/docling edge case)
        (
            MagicMock(
                metadata=MagicMock(
                    title="Callable Title",
                    authors=lambda: ["Callable Author"],
                    creation_date=lambda: "2024-01-01",
                    language="de",
                ),
                num_pages=lambda: 10,
            ),
            MagicMock(),
            "c.pdf",
            {
                "title": "Callable Title",
                "author": "Callable Author",
                "created_at": "2024-01-01",
                "pages": 10,
                "language": "de",
            },
        ),
    ],
)
def test_file_metadata_extraction_logic(
    doc_mock, result_mock, file_path, expected_meta
):
    # Mocking behaviors for callables if necessary, but MagicMock usually handles attrs.
    # For lambda simulation in MagicMock, we might need side_effect or return_value if it's called.

    # Adjust mocks for list vs string authors if logic expects specific types
    if expected_meta["author"] == "Callable Author":
        # If the code expects a list, make sure the mock returns it when accessed or is iterable?
        # The current implementation checks: isinstance(doc.metadata.authors, list)
        # If it's a method that returns a list, the current code might fail if it doesn't call it first.
        # Let's see how the code handles it.
        pass

    meta = extract_metadata_from_doc(doc_mock, result_mock, file_path)
    assert meta == expected_meta


# --- Web Metadata Tests ---


@pytest.mark.parametrize(
    "crawl_result_mock, url, expected_meta",
    [
        # Case 1: Standard markdown title and links
        (
            MagicMock(
                markdown="# Web Title\nSome content",
                links={
                    "internal": [
                        {"href": "http://e.com/1"},
                        {"href": "http://e.com/2"},
                    ],
                    "external": [],
                },
                url="http://e.com/page",
            ),
            "http://e.com/page",
            {"title": "Web Title", "path": "page", "links_count": 2},
        ),
        # Case 2: No markdown title
        (
            MagicMock(
                markdown="No header here", links={}, url="http://e.com/nested/path"
            ),
            "http://e.com/nested/path",
            {"title": "", "path": "nested > path", "links_count": 0},
        ),
        # Case 3: Markdown links extraction (llms.txt style)
        (
            MagicMock(
                markdown="# Index\n[Link 1](subpage) [External](http://google.com)",
                links={},
                url="http://e.com/",
            ),
            "http://e.com/",
            {
                "title": "Index",
                "path": "",
                "links_count": 1,
            },  # Only internal 'subpage' -> 'http://e.com/subpage'
        ),
    ],
)
def test_web_metadata_extraction_logic(crawl_result_mock, url, expected_meta):
    meta = extract_web_metadata(crawl_result_mock, url)

    assert meta["title"] == expected_meta["title"]
    assert meta["path"] == expected_meta["path"]
    if "links_count" in expected_meta:
        assert len(meta["links"]) == expected_meta["links_count"]
