interface SearchResult {
    id: string;
    title: string;
    url: string;
    content: string;
    score: number;
}

interface SearchResponse {
    semantic_results: SearchResult[];
    bm25_results: SearchResult[];
    total: number;
    duration: string;
}

interface SummaryResponse {
    summary: string;
    model?: string;
    duration: string;
}

const searchInput = document.getElementById('search-input') as HTMLInputElement;
const resultsContainer = document.getElementById('results-container') as HTMLElement;
const summaryText = document.getElementById('summary-text') as HTMLElement;
const summaryStatus = document.getElementById('summary-status') as HTMLElement;

const searchIcon = document.querySelector('.search-icon') as HTMLElement;

if (searchInput && resultsContainer) {
    // Search on Enter key
    searchInput.addEventListener('keydown', (e: KeyboardEvent) => {
        if (e.key === 'Enter') {
            const query = searchInput.value.trim();
            if (query.length > 0) {
                performSearch(query);
            }
        }
    });
}

if (searchIcon && searchInput) {
    // Search on Icon click
    searchIcon.addEventListener('click', () => {
        const query = searchInput.value.trim();
        if (query.length > 0) {
            performSearch(query);
        }
    });
}

async function performSearch(query: string): Promise<void> {
    if (!resultsContainer) return;

    try {
        setSummaryLoading();
        const response = await fetch(`/api/search?q=${encodeURIComponent(query)}`);
        if (!response.ok) {
            throw new Error('Network response was not ok');
        }
        const data: SearchResponse = await response.json();
        console.log('API Response:', data);
        console.log('Semantic results:', data.semantic_results?.length || 0);
        console.log('BM25 results:', data.bm25_results?.length || 0);
        renderResults(data.semantic_results, data.bm25_results, data.total, data.duration);
        void fetchSummary(query, data.semantic_results, data.bm25_results);
    } catch (error) {
        console.error('Error fetching search results:', error);
        resultsContainer.innerHTML = '<div class="no-results">An error occurred while searching.</div>';
        setSummaryError();
    }
}

function setSummaryLoading(): void {
    if (summaryStatus) {
        summaryStatus.textContent = 'Generating';
    }
    if (summaryText) {
        summaryText.textContent = 'Generating an AI summary based on the results...';
        summaryText.classList.add('is-loading');
    }
}

function setSummaryError(): void {
    if (summaryStatus) {
        summaryStatus.textContent = 'Unavailable';
    }
    if (summaryText) {
        summaryText.textContent = 'Summary unavailable right now. Please try again later.';
        summaryText.classList.add('is-loading');
    }
}

function setSummaryReady(summary: string): void {
    if (summaryStatus) {
        summaryStatus.textContent = 'Ready';
    }
    if (summaryText) {
        summaryText.textContent = summary;
        summaryText.classList.remove('is-loading');
    }
}

function buildSummaryPayload(semanticResults: SearchResult[], bm25Results: SearchResult[]): SearchResult[] {
    const combined = [...(semanticResults || []), ...(bm25Results || [])];
    const trimmed: SearchResult[] = [];
    const seen = new Set<string>();

    for (const result of combined) {
        if (trimmed.length >= 6) break;
        if (seen.has(result.url)) continue;
        seen.add(result.url);
        trimmed.push(result);
    }

    return trimmed;
}

async function fetchSummary(query: string, semanticResults: SearchResult[], bm25Results: SearchResult[]): Promise<void> {
    if (!summaryText || !summaryStatus) return;

    const payload = {
        query,
        results: buildSummaryPayload(semanticResults, bm25Results).map((result) => ({
            title: result.title,
            url: result.url,
            content: result.content,
        })),
    };

    try {
        const response = await fetch('/api/summary', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(payload),
        });
        if (!response.ok) {
            throw new Error('Summary response was not ok');
        }
        const data: SummaryResponse = await response.json();
        setSummaryReady(data.summary);
    } catch (error) {
        console.error('Error fetching AI summary:', error);
        setSummaryError();
    }
}

function renderResults(semanticResults: SearchResult[], bm25Results: SearchResult[], total: number, duration: string): void {
    if (!resultsContainer) return;
    resultsContainer.innerHTML = '';

    if ((!semanticResults || semanticResults.length === 0) && (!bm25Results || bm25Results.length === 0)) {
        resultsContainer.innerHTML = '<div class="no-results">No results found in the cosmos.</div>';
        return;
    }

    // Display stats
    const stats = document.createElement('div');
    stats.className = 'search-stats';
    stats.textContent = `Found ${total} results in ${duration}`;
    resultsContainer.appendChild(stats);

    // Create two columns container
    const columnsContainer = document.createElement('div');
    columnsContainer.className = 'results-grid';

    // BM25 Results Column
    const bm25Column = document.createElement('div');
    bm25Column.className = 'results-column';
    const bm25Header = document.createElement('h2');
    bm25Header.textContent = 'BM25 Results';
    bm25Column.appendChild(bm25Header);

    if (bm25Results && bm25Results.length > 0) {
        bm25Results.forEach((result, index) => {
            bm25Column.appendChild(createResultCard(result, index));
        });
    } else {
        const noResults = document.createElement('div');
        noResults.className = 'no-results';
        noResults.textContent = 'No BM25 results';
        bm25Column.appendChild(noResults);
    }

    // Semantic Results Column
    const semanticColumn = document.createElement('div');
    semanticColumn.className = 'results-column';
    const semanticHeader = document.createElement('h2');
    semanticHeader.textContent = 'Semantic Search Results';
    semanticColumn.appendChild(semanticHeader);

    if (semanticResults && semanticResults.length > 0) {
        semanticResults.forEach((result, index) => {
            semanticColumn.appendChild(createResultCard(result, index));
        });
    } else {
        const noResults = document.createElement('div');
        noResults.className = 'no-results';
        noResults.textContent = 'No semantic results';
        semanticColumn.appendChild(noResults);
    }

    columnsContainer.appendChild(bm25Column);
    columnsContainer.appendChild(semanticColumn);
    resultsContainer.appendChild(columnsContainer);
}

function createResultCard(result: SearchResult, index: number): HTMLElement {
    const card = document.createElement('div');
    card.className = 'result-card';
    (card as HTMLElement).style.animationDelay = `${index * 0.05}s`;

    const header = document.createElement('div');
    header.className = 'result-header';

    const titleLink = document.createElement('a');
    titleLink.className = 'result-title';
    titleLink.href = result.url;
    titleLink.target = '_blank';
    titleLink.textContent = result.title || `Document #${result.id}`;

    const score = document.createElement('span');
    score.className = 'result-score';
    score.textContent = `Score: ${result.score.toFixed(2)}`;
    header.appendChild(titleLink);
    header.appendChild(score);

    const url = document.createElement('div');
    url.className = 'result-url';
    url.textContent = result.url;

    const content = document.createElement('div');
    content.className = 'result-content';
    content.textContent = result.content;

    card.appendChild(header);
    card.appendChild(url);
    card.appendChild(content);
    return card;
}
