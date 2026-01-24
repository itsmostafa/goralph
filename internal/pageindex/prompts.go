package pageindex

// LLM prompt templates for PageIndex operations.
// These prompts are based on the Python PageIndex implementation.

// TOCDetectorPrompt asks the LLM to determine if a page contains TOC content.
const TOCDetectorPrompt = `You are an expert in analyzing document pages. You are given the text of a page in a PDF document. Your task is to determine whether this page contains a Table of Contents (TOC).

A Table of Contents typically:
- Contains a list of section/chapter titles
- May include page numbers
- Has a hierarchical structure (chapters, sections, subsections)
- Appears near the beginning of a document

Page Text:
%s

Respond in JSON format:
{
  "thinking": "<your reasoning>",
  "toc_detected": "<yes or no>"
}`

// PageIndexGivenPrompt asks whether page numbers are included in the TOC.
const PageIndexGivenPrompt = `You are an expert in analyzing document structure. You are given a Table of Contents from a PDF document. Your task is to determine whether page numbers are explicitly given in the Table of Contents.

Table of Contents:
%s

Respond in JSON format:
{
  "thinking": "<your reasoning>",
  "page_index_given_in_toc": "<yes or no>"
}`

// TOCTransformPrompt converts raw TOC text into structured JSON format.
const TOCTransformPrompt = `You are an expert in parsing document structures. You are given a Table of Contents from a PDF document. Your task is to parse it into a structured JSON format.

For each entry, extract:
- structure: The hierarchical index (e.g., "1", "1.1", "1.2.3"). Use "0" for unnumbered items like "Preface"
- title: The section title
- page: The page number if given (as integer), or null if not present

Table of Contents:
%s

Respond in JSON format:
{
  "table_of_contents": [
    {"structure": "1", "title": "Introduction", "page": 5},
    {"structure": "1.1", "title": "Background", "page": 7},
    ...
  ]
}`

// TOCTransformContinuePrompt asks the LLM to continue an incomplete TOC transformation.
const TOCTransformContinuePrompt = `Continue parsing the Table of Contents from where you left off. Keep the same JSON format.`

// VerifyTOCEntryPrompt asks the LLM to verify if a TOC entry appears on a page.
const VerifyTOCEntryPrompt = `You are an expert in verifying document structure. You are given:
1. A section title from a Table of Contents
2. The text of a page where the section should appear

Your task is to determine whether the section title appears on this page. The title may have slight variations in spacing or formatting.

Section Title: %s
Expected Page Number: %d

Page Text:
%s

Respond in JSON format:
{
  "thinking": "<your reasoning>",
  "answer": "<yes or no>"
}`

// VerifyBatchPrompt verifies multiple TOC entries at once.
const VerifyBatchPrompt = `You are an expert in verifying document structure. You are given:
1. A list of section titles from a Table of Contents
2. A list of page texts

Your task is to verify which sections appear on their expected pages. For each entry, check if the title (or a close variation) appears on the corresponding page.

Entries to verify:
%s

Page Texts:
%s

Respond in JSON format:
{
  "results": [
    {"list_index": 0, "answer": "yes", "title": "...", "page_number": 5},
    {"list_index": 1, "answer": "no", "title": "...", "page_number": 10},
    ...
  ]
}`

// StartCheckPrompt determines if a section starts at the beginning of a page.
const StartCheckPrompt = `You are an expert in analyzing document layout. You are given:
1. A section title
2. The text of a page

Your task is to determine whether the section title appears at or near the BEGINNING of the page (first ~200 characters), indicating the section starts on this page.

Section Title: %s

Page Text (first 500 characters):
%s

Respond in JSON format:
{
  "thinking": "<your reasoning>",
  "start_begin": "<yes or no>"
}`

// CompletionCheckPrompt determines if a TOC transformation is complete.
const CompletionCheckPrompt = `You are given a partial JSON response that was parsing a Table of Contents. Determine if the parsing appears to be complete or if there are likely more entries to parse.

Partial Response:
%s

Original TOC Content:
%s

Respond in JSON format:
{
  "thinking": "<your reasoning>",
  "completed": "<yes or no>"
}`

// GenerateTOCPrompt generates a TOC when none exists in the document.
const GenerateTOCPrompt = `You are an expert in analyzing document structure. You are given pages from a PDF document that does not have an explicit Table of Contents. Your task is to generate a Table of Contents based on the document's structure.

Look for:
- Chapter/section headers
- Clear topic changes
- Numbered sections
- Bold or emphasized titles

Document Pages:
%s

Generate a Table of Contents in JSON format:
{
  "table_of_contents": [
    {"structure": "1", "title": "Introduction", "physical_index": 1},
    {"structure": "1.1", "title": "Background", "physical_index": 3},
    ...
  ]
}

Use physical_index for the actual page number where each section begins.`

// FixTOCPrompt asks the LLM to fix incorrect page numbers in a TOC.
const FixTOCPrompt = `You are an expert in correcting document structure. You are given:
1. A Table of Contents with some incorrect page numbers
2. Verification results showing which entries are incorrect
3. Page texts around the incorrect entries

Your task is to fix the incorrect page numbers by searching for the correct pages where the sections actually appear.

Current TOC:
%s

Incorrect Entries (list_index, expected_page):
%s

Page Texts (page_number: text):
%s

Respond with the corrected entries in JSON format:
{
  "corrections": [
    {"list_index": 0, "corrected_page": 8, "thinking": "..."},
    ...
  ]
}`

// SplitNodePrompt generates child entries when splitting a large node.
const SplitNodePrompt = `You are an expert in analyzing document structure. A section of a document is too large and needs to be split into smaller subsections. You are given the text of the section.

Generate a list of subsections based on the content structure. Look for:
- Topic changes
- Natural paragraph groupings
- Subheadings within the text
- Logical content divisions

Section Title: %s
Section Text:
%s

Generate subsections in JSON format:
{
  "subsections": [
    {"title": "Subsection Title", "start_position": 0, "end_position": 500},
    ...
  ]
}

Positions are character offsets within the section text.`

// SummaryPrompt generates a summary for a document section.
const SummaryPrompt = `You are an expert in summarizing documents. You are given a part of a document. Generate a concise summary that captures the main points.

Section Title: %s
Section Text:
%s

Generate a summary in 2-3 sentences that captures the key information. Respond with just the summary text, no JSON formatting.`

// DocumentDescriptionPrompt generates a one-sentence description for the entire document.
const DocumentDescriptionPrompt = `You are an expert in generating document descriptions. You are given the structure of a document. Generate a one-sentence description that distinguishes this document from others.

Document Structure:
%s

Respond with just the description, no other text.`
