"use strict";
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
const searchInput = document.getElementById('search-input');
const resultsContainer = document.getElementById('results-container');
const summaryText = document.getElementById('summary-text');
const summaryStatus = document.getElementById('summary-status');
const searchIcon = document.querySelector('.search-icon');
if (searchInput && resultsContainer) {
    searchInput.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') {
            const query = searchInput.value.trim();
            if (query.length > 0) {
                performSearch(query);
            }
        }
    });
}
if (searchIcon && searchInput) {
    searchIcon.addEventListener('click', () => {
        const query = searchInput.value.trim();
        if (query.length > 0) {
            performSearch(query);
        }
    });
}
function performSearch(query) {
    return __awaiter(this, void 0, void 0, function* () {
        var _a, _b;
        if (!resultsContainer)
            return;
        try {
            setSummaryLoading();
            const response = yield fetch(`/api/search?q=${encodeURIComponent(query)}`);
            if (!response.ok) {
                throw new Error('Network response was not ok');
            }
            const data = yield response.json();
            console.log('API Response:', data);
            console.log('Semantic results:', ((_a = data.semantic_results) === null || _a === void 0 ? void 0 : _a.length) || 0);
            console.log('BM25 results:', ((_b = data.bm25_results) === null || _b === void 0 ? void 0 : _b.length) || 0);
            renderResults(data.semantic_results, data.bm25_results, data.total, data.duration);
            void fetchSummary(query, data.semantic_results, data.bm25_results);
        }
        catch (error) {
            console.error('Error fetching search results:', error);
            resultsContainer.innerHTML = '<div class="no-results">An error occurred while searching.</div>';
            setSummaryError();
        }
    });
}
function setSummaryLoading() {
    if (summaryStatus) {
        summaryStatus.textContent = 'Generating';
    }
    if (summaryText) {
        summaryText.textContent = 'Generating an AI summary based on the results...';
        summaryText.classList.add('is-loading');
    }
}
function setSummaryError() {
    if (summaryStatus) {
        summaryStatus.textContent = 'Unavailable';
    }
    if (summaryText) {
        summaryText.textContent = 'Summary unavailable right now. Please try again later.';
        summaryText.classList.add('is-loading');
    }
}
function setSummaryReady(summary) {
    if (summaryStatus) {
        summaryStatus.textContent = 'Ready';
    }
    if (summaryText) {
        summaryText.textContent = summary;
        summaryText.classList.remove('is-loading');
    }
}
function buildSummaryPayload(semanticResults, bm25Results) {
    const combined = [...(semanticResults || []), ...(bm25Results || [])];
    const trimmed = [];
    const seen = new Set();
    for (const result of combined) {
        if (trimmed.length >= 6)
            break;
        if (seen.has(result.url))
            continue;
        seen.add(result.url);
        trimmed.push(result);
    }
    return trimmed;
}
function fetchSummary(query, semanticResults, bm25Results) {
    return __awaiter(this, void 0, void 0, function* () {
        if (!summaryText || !summaryStatus)
            return;
        const payload = {
            query,
            results: buildSummaryPayload(semanticResults, bm25Results).map((result) => ({
                title: result.title,
                url: result.url,
                content: result.content,
            })),
        };
        try {
            const response = yield fetch('/api/summary', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(payload),
            });
            if (!response.ok) {
                throw new Error('Summary response was not ok');
            }
            const data = yield response.json();
            setSummaryReady(data.summary);
        }
        catch (error) {
            console.error('Error fetching AI summary:', error);
            setSummaryError();
        }
    });
}
function renderResults(semanticResults, bm25Results, total, duration) {
    if (!resultsContainer)
        return;
    resultsContainer.innerHTML = '';
    if ((!semanticResults || semanticResults.length === 0) && (!bm25Results || bm25Results.length === 0)) {
        resultsContainer.innerHTML = '<div class="no-results">No results found in the cosmos.</div>';
        return;
    }
    const stats = document.createElement('div');
    stats.className = 'search-stats';
    stats.textContent = `Found ${total} results in ${duration}`;
    resultsContainer.appendChild(stats);
    const columnsContainer = document.createElement('div');
    columnsContainer.className = 'results-grid';
    const bm25Column = document.createElement('div');
    bm25Column.className = 'results-column';
    const bm25Header = document.createElement('h2');
    bm25Header.textContent = 'BM25 Results';
    bm25Column.appendChild(bm25Header);
    if (bm25Results && bm25Results.length > 0) {
        bm25Results.forEach((result, index) => {
            bm25Column.appendChild(createResultCard(result, index));
        });
    }
    else {
        const noResults = document.createElement('div');
        noResults.className = 'no-results';
        noResults.textContent = 'No BM25 results';
        bm25Column.appendChild(noResults);
    }
    const semanticColumn = document.createElement('div');
    semanticColumn.className = 'results-column';
    const semanticHeader = document.createElement('h2');
    semanticHeader.textContent = 'Semantic Search Results';
    semanticColumn.appendChild(semanticHeader);
    if (semanticResults && semanticResults.length > 0) {
        semanticResults.forEach((result, index) => {
            semanticColumn.appendChild(createResultCard(result, index));
        });
    }
    else {
        const noResults = document.createElement('div');
        noResults.className = 'no-results';
        noResults.textContent = 'No semantic results';
        semanticColumn.appendChild(noResults);
    }
    columnsContainer.appendChild(bm25Column);
    columnsContainer.appendChild(semanticColumn);
    resultsContainer.appendChild(columnsContainer);
}
function createResultCard(result, index) {
    const card = document.createElement('div');
    card.className = 'result-card';
    card.style.animationDelay = `${index * 0.05}s`;
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
