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
        if (!resultsContainer)
            return;
        try {
            const response = yield fetch(`/api/search?q=${encodeURIComponent(query)}`);
            if (!response.ok) {
                throw new Error('Network response was not ok');
            }
            const data = yield response.json();
            renderResults(data.results, data.total, data.duration);
        }
        catch (error) {
            console.error('Error fetching search results:', error);
            resultsContainer.innerHTML = '<div class="no-results">An error occurred while searching.</div>';
        }
    });
}
function renderResults(results, total, duration) {
    if (!resultsContainer)
        return;
    resultsContainer.innerHTML = '';
    if (!results || results.length === 0) {
        resultsContainer.innerHTML = '<div class="no-results">No results found in the cosmos.</div>';
        return;
    }
    const stats = document.createElement('div');
    stats.className = 'search-stats';
    stats.textContent = `Found ${total} results in ${duration}`;
    resultsContainer.appendChild(stats);
    results.forEach((result, index) => {
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
        const truncScore = Math.trunc(result.score);
        score.className = 'result-score';
        score.textContent = `Score: ${truncScore}`;
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
        resultsContainer.appendChild(card);
    });
}
