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

const searchInput = document.getElementById('search-input') as HTMLInputElement;
const resultsContainer = document.getElementById('results-container') as HTMLElement;

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
        const response = await fetch(`/api/search?q=${encodeURIComponent(query)}`);
        if (!response.ok) {
            throw new Error('Network response was not ok');
        }
        const data: SearchResponse = await response.json();
        renderResults(data.semantic_results, data.bm25_results, data.total, data.duration);
    } catch (error) {
        console.error('Error fetching search results:', error);
        resultsContainer.innerHTML = '<div class="no-results">An error occurred while searching.</div>';
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
