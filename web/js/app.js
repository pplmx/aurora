// auroraHeaders returns fetch headers that authenticate the same-origin API
// calls. The API gateway injects the resolved key into served HTML as
// window.AURORA_API_KEY; without it every /api/v1 call would return 401.
function auroraHeaders(extra) {
    return Object.assign({ 'X-API-Key': window.AURORA_API_KEY || '' }, extra || {});
}

// A single, fixed, visible banner for API failures (round-97 TASK-124).
// List GETs previously stored the {error, code} envelope into array state on a
// missing/expired key, rendering blank pages with zero feedback. All fetch
// sites now go through apiFetch, which checks res.ok and surfaces the failure
// here in the DOM instead of silently swallowing it.
(function () {
    let banner = null;
    function ensureBanner() {
        if (banner) return banner;
        banner = document.createElement('div');
        banner.id = 'api-error-banner';
        banner.style.cssText = 'position:fixed;top:0;left:0;right:0;z-index:9999;background:#c0392b;' +
            'color:#fff;padding:10px 16px;text-align:center;font:13px/1.4 system-ui,sans-serif;' +
            'display:none;box-shadow:0 2px 6px rgba(0,0,0,.3);';
        document.body.appendChild(banner);
        return banner;
    }
    window.showApiError = function (msg) {
        const b = ensureBanner();
        if (!msg) { b.style.display = 'none'; return; }
        b.textContent = 'API error: ' + msg;
        b.style.display = 'block';
    };
})();

// apiFetch performs a same-origin API call with the injected key, checks
// res.ok, and on a non-2xx response extracts the API's {error, code} envelope
// (falling back to the HTTP status), shows it in the shared banner and throws.
// Callers keep list state valid (e.g. []) in their catch so a failure renders
// an empty/error UI instead of a non-array that Alpine's x-for rejects.
async function apiFetch(path, options) {
    const res = await fetch(path, Object.assign({ headers: auroraHeaders() }, options || {}));
    if (!res.ok) {
        let msg = 'HTTP ' + res.status;
        try {
            const body = await res.json();
            if (body && (body.error || body.message)) msg = String(body.error || body.message) + ' (' + res.status + ')';
        } catch (_) { /* non-JSON error body */ }
        window.showApiError(msg);
        throw new Error(msg);
    }
    // A 2xx response means the API recovered; dismiss any sticky failure
    // banner a transient earlier error left behind (round-102 TASK-141).
    window.showApiError('');
    return res;
}

// startPolling re-runs target.refresh() every intervalMs while the page is
// alive, for surfaces whose data changes without user action (the dashboard
// overview, Oracle sources which the background scheduler updates). The
// running flag prevents overlapping refreshes when one takes longer than the
// interval.
function startPolling(target, intervalMs) {
    let running = false;
    setInterval(async () => {
        if (running || typeof target.refresh !== 'function') return;
        running = true;
        try {
            await target.refresh();
        } finally {
            running = false;
        }
    }, intervalMs);
}

function lotteryApp() {
    return {
        participants: '',
        seed: '',
        count: 1,
        result: '',
        history: [],
        loading: true,
        verifyId: '',
        verifyResult: '',
        async init() {
            await this.loadHistory();
        },
        async verifyDraw() {
            try {
                const res = await apiFetch('/api/v1/lottery/' + encodeURIComponent(this.verifyId) + '/verify');
                this.verifyResult = JSON.stringify(await res.json(), null, 2);
            } catch (e) {
                this.verifyResult = 'Error: ' + e.message;
            }
        },
        async loadHistory() {
            this.loading = true;
            try {
                const res = await apiFetch('/api/v1/lottery/history');
                const data = await res.json();
                this.history = Array.isArray(data) ? data : [];
            } catch (e) {
                this.history = [];
            }
            this.loading = false;
        },
        async createLottery() {
            try {
                const res = await apiFetch('/api/v1/lottery/create', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        participants: this.participants.split(',').map(p => p.trim()).filter(Boolean).join(','),
                        seed: this.seed || undefined,
                        winner_count: parseInt(this.count)
                    })
                });
                const data = await res.json();
                this.result = JSON.stringify(data, null, 2);
                await this.loadHistory();
                this.participants = '';
                this.seed = '';
            } catch (e) {
                this.result = 'Error: ' + e.message;
            }
        }
    };
}

function dashboardApp() {
    return {
        stats: { lotteries: 0, votes: 0, candidates: 0, sessions: 0, integrity: '-', oracle: '-' },
        activity: [],
        loading: true,
        async init() {
            await this.refresh();
            this.loading = false;
            // The dashboard is a live overview; re-poll so a freshly recorded
            // vote/lottery/feed-health shows up without a manual reload (ISS-125).
            startPolling(this, 15000);
        },
        // refresh re-pulls every stat card and rebuilds the Recent Activity
        // list; also exposed as the header "↻ Refresh" button. It does NOT
        // blank the stats grid up front: each loader overwrites its own card
        // as its result arrives, so a live surface keeps its previous snapshot
        // instead of dipping to 0/- on every 15s poll (TASK-151). The activity
        // entries are collected per-loader and swapped in once all loads
        // finish, so the list no longer clears then re-flashes each cycle.
        async refresh() {
            const activity = await Promise.all([
                this.loadLotteries(),
                this.loadVoting(),
                this.loadBlockchain(),
                this.loadOracleHealth()
            ]);
            this.activity = [].concat(...activity);
        },
        async loadLotteries() {
            const entries = [];
            try {
                const res = await apiFetch('/api/v1/lottery/history');
                const data = await res.json();
                if (Array.isArray(data)) {
                    this.stats.lotteries = data.length;
                    data.slice(0, 10).forEach(l => {
                        entries.push({
                            key: 'lot-' + (l.id || Math.random()),
                            title: 'Lottery ' + (l.id || ''),
                            detail: (l.winners || []).length + ' winner(s)'
                        });
                    });
                }
            } catch (e) {
                console.error(e);
            }
            return entries;
        },
        async loadVoting() {
            // Each stat group has its own try/catch so a transient failure on
            // one endpoint (e.g. sessions 500) blanks only its own card instead
            // of the shared Promise.all dropping the sibling stats too (TASK-147).
            const groups = await Promise.all([this.loadCandidatesStats(), this.loadSessionsStats()]);
            return [].concat(...groups);
        },
        async loadCandidatesStats() {
            try {
                const res = await apiFetch('/api/v1/voting/candidates');
                const candidates = await res.json();
                if (Array.isArray(candidates)) {
                    this.stats.candidates = candidates.length;
                    // "Total votes" has no dedicated endpoint; derive it from
                    // each candidate's real vote_count.
                    this.stats.votes = candidates.reduce((sum, c) => sum + (c.vote_count || 0), 0);
                }
            } catch (e) {
                console.error(e);
            }
            return []; // candidates contribute no activity rows
        },
        async loadSessionsStats() {
            const entries = [];
            try {
                const res = await apiFetch('/api/v1/voting/sessions');
                const sessions = await res.json();
                if (Array.isArray(sessions)) {
                    this.stats.sessions = sessions.length;
                    sessions.slice(0, 10).forEach(s => {
                        entries.push({
                            key: 'sess-' + s.id,
                            title: 'Session: ' + s.title,
                            detail: 'Status: ' + s.status + ' · ' + (s.candidates || []).length + ' candidate(s)'
                        });
                    });
                }
            } catch (e) {
                console.error(e);
            }
            return entries;
        },
        async loadBlockchain() {
            try {
                const report = await (await apiFetch('/api/v1/blockchain/verify')).json();
                this.stats.integrity = report.valid ? 'OK' : 'BROKEN';
            } catch (e) {
                this.stats.integrity = '?';
                console.error(e);
            }
            return []; // the ledger card adds no activity row
        },
        async loadOracleHealth() {
            try {
                const feeds = await (await apiFetch('/api/v1/oracle/health')).json();
                if (!Array.isArray(feeds) || feeds.length === 0) { this.stats.oracle = '-'; return []; }
                const healthy = feeds.filter(f => f.successes > 0 && f.failures === 0).length;
                const failed = feeds.filter(f => f.failures > 0).length;
                this.stats.oracle = healthy + ' OK' + (failed ? ' · ' + failed + ' fail' : '');
            } catch (e) {
                this.stats.oracle = '?';
                console.error(e);
            }
            return []; // feed health adds no activity row
        }
    };
}

// blockchainApp drives the /blockchain.html ledger integrity page (v1.26). It
// calls the protected GET /api/v1/blockchain/verify endpoint and renders the
// IntegrityReport (valid, chain length, first broken index + reason). It goes
// through apiFetch so a down/unauthorized API surfaces in the shared banner
// like every other web surface (TASK-124/141 contract), not just inline.
function blockchainApp() {
    return {
        report: null,
        error: '',
        loading: false,
        async verify() {
            this.loading = true;
            this.error = '';
            try {
                const res = await apiFetch('/api/v1/blockchain/verify');
                this.report = await res.json();
            } catch (e) {
                this.error = 'Error: ' + e.message;
            } finally {
                this.loading = false;
            }
        }
    };
}

// nftApp drives the /nft.html page (v1.27): mint, list-by-owner, get,
// transfer and burn via the protected /api/v1/nft REST endpoints.
function nftApp() {
    return {
        name: '',
        description: '',
        imageUrl: '',
        tokenUri: '',
        creator: '',
        owner: '',
        from: '',
        to: '',
        privateKey: '',
        id: '',
        mintResult: '',
        listResult: '',
        getResult: '',
        transferResult: '',
        burnResult: '',
        historyId: '',
        historyResult: '',
        async mint() {
            try {
                const res = await apiFetch('/api/v1/nft/mint', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        name: this.name,
                        description: this.description || undefined,
                        image_url: this.imageUrl || undefined,
                        token_uri: this.tokenUri || undefined,
                        creator: this.creator
                    })
                });
                const data = await res.json();
                this.mintResult = JSON.stringify(data, null, 2);
                // Advance the shared Get/Transfer/Burn/History context to the
                // freshly minted NFT so the next step needs no manual copy from
                // the JSON result block (TASK-150).
                if (data && data.id) this.id = data.id;
                if (data && data.owner) this.owner = data.owner;
            } catch (e) {
                this.mintResult = 'Error: ' + e.message;
            }
        },
        async list() {
            try {
                const res = await apiFetch('/api/v1/nft/list?owner=' + encodeURIComponent(this.owner));
                this.listResult = await this.text(res);
            } catch (e) {
                this.listResult = 'Error: ' + e.message;
            }
        },
        async get() {
            try {
                const res = await apiFetch('/api/v1/nft/' + encodeURIComponent(this.id));
                this.getResult = await this.text(res);
            } catch (e) {
                this.getResult = 'Error: ' + e.message;
            }
        },
        async history() {
            try {
                const res = await apiFetch('/api/v1/nft/' + encodeURIComponent(this.historyId) + '/history');
                this.historyResult = JSON.stringify(await res.json(), null, 2);
            } catch (e) {
                this.historyResult = 'Error: ' + e.message;
            }
        },
        async transfer() {
            try {
                this.transferResult = await this.post('/api/v1/nft/transfer', {
                    nft_id: this.id,
                    from: this.from,
                    to: this.to,
                    private_key: this.privateKey
                });
            } catch (e) {
                this.transferResult = 'Error: ' + e.message;
            }
        },
        async burn() {
            try {
                this.burnResult = await this.post('/api/v1/nft/burn', {
                    nft_id: this.id,
                    owner: this.owner,
                    private_key: this.privateKey
                });
            } catch (e) {
                this.burnResult = 'Error: ' + e.message;
            }
        },
        async post(url, body) {
            const res = await apiFetch(url, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(body)
            });
            return this.text(res);
        },
        async text(res) {
            const data = await res.json();
            return JSON.stringify(data, null, 2);
        }
    };
}

function votingApp() {
    return {
        voterName: '',
        voterResult: '',
        voterPrivateKey: '',
        candName: '',
        candParty: '',
        candProgram: '',
        candResult: '',
        sessionTitle: '',
        sessionDesc: '',
        sessionStart: '',
        sessionEnd: '',
        sessionCandidateIds: [],
        sessionResult: '',
        voteSessionId: '',
        voteCandidateId: '',
        voteVoterPub: '',
        votePriv: '',
        voteResult: '',
        resultsSessionId: '',
        results: null,
        resultsError: '',
        controlSessionId: '',
        controlResult: '',
        candidates: [],
        sessions: [],
        loading: true,
        async init() {
            await Promise.all([this.loadCandidates(), this.loadSessions()]);
            this.loading = false;
        },
        async loadCandidates() {
            try {
                const res = await apiFetch('/api/v1/voting/candidates');
                const data = await res.json();
                this.candidates = Array.isArray(data) ? data : [];
            } catch (e) {
                this.candidates = [];
                console.error(e);
            }
        },
        async loadSessions() {
            try {
                const res = await apiFetch('/api/v1/voting/sessions');
                const data = await res.json();
                this.sessions = Array.isArray(data) ? data : [];
            } catch (e) {
                this.sessions = [];
                console.error(e);
            }
        },
        candidateNames(ids) {
            return (ids || []).map(id => {
                const c = this.candidates.find(x => x.id === id);
                return c ? c.name : id;
            });
        },
        get currentSessionCandidates() {
            const s = this.sessions.find(x => x.id === this.voteSessionId);
            if (!s) return [];
            return (s.candidates || [])
                .map(id => this.candidates.find(c => c.id === id))
                .filter(Boolean);
        },
        async registerVoter() {
            try {
                const res = await apiFetch('/api/v1/voting/register/voter', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ name: this.voterName })
                });
                const data = await res.json();
                this.voterResult = 'Voter registered as ' + data.name + ' (public key: ' + data.public_key + ')';
                this.voterPrivateKey = data.private_key;
                this.voteVoterPub = data.public_key;
                this.votePriv = data.private_key;
                this.voterName = '';
            } catch (e) {
                this.voterResult = 'Error: ' + e.message;
            }
        },
        async registerCandidate() {
            try {
                const res = await apiFetch('/api/v1/voting/register/candidate', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        name: this.candName,
                        party: this.candParty,
                        program: this.candProgram
                    })
                });
                const data = await res.json();
                this.candResult = 'Candidate "' + data.name + '" registered (id: ' + data.id + ')';
                this.candName = '';
                this.candParty = '';
                this.candProgram = '';
                await this.loadCandidates();
            } catch (e) {
                this.candResult = 'Error: ' + e.message;
            }
        },
        async createSession() {
            try {
                const res = await apiFetch('/api/v1/voting/session', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        title: this.sessionTitle,
                        description: this.sessionDesc,
                        candidate_ids: this.sessionCandidateIds,
                        start_time: parseInt(this.sessionStart),
                        end_time: parseInt(this.sessionEnd)
                    })
                });
                const data = await res.json();
                this.sessionResult = 'Session "' + data.title + '" created (id: ' + data.id + ')';
                this.sessionTitle = '';
                this.sessionDesc = '';
                this.sessionStart = '';
                this.sessionEnd = '';
                this.sessionCandidateIds = [];
                await this.loadSessions();
            } catch (e) {
                this.sessionResult = 'Error: ' + e.message;
            }
        },
        async castVote() {
            try {
                const res = await apiFetch('/api/v1/voting/vote', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        voter_public_key: this.voteVoterPub,
                        candidate_id: this.voteCandidateId,
                        private_key: this.votePriv,
                        session_id: this.voteSessionId
                    })
                });
                const data = await res.json();
                this.voteResult = 'Vote recorded (id: ' + data.id + ', block height: ' + data.block_height + ')';
                await this.loadCandidates();
            } catch (e) {
                this.voteResult = 'Error: ' + e.message;
            }
        },
        async loadResults() {
            try {
                const res = await apiFetch('/api/v1/voting/results/' + encodeURIComponent(this.resultsSessionId));
                const data = await res.json();
                this.results = data;
                this.resultsError = '';
            } catch (e) {
                this.results = null;
                this.resultsError = 'Error: ' + e.message;
            }
        },
        async startSession() {
            this.controlResult = '…starting';
            await this.sessionAction('/api/v1/voting/session/' + encodeURIComponent(this.controlSessionId) + '/start');
        },
        async endSession() {
            this.controlResult = '…ending';
            await this.sessionAction('/api/v1/voting/session/' + encodeURIComponent(this.controlSessionId) + '/end');
        },
        async sessionAction(url) {
            try {
                const res = await apiFetch(url, { method: 'POST' });
                const data = await res.json();
                this.controlResult = 'Session status: ' + (data.status || '?');
                await this.loadSessions();
            } catch (e) {
                this.controlResult = 'Error: ' + e.message;
            }
        }
    };
}

function tokenApp() {
    return {
        name: '', symbol: '', supply: '', createResult: '',
        tokenId: '', owner: '', balance: '',
        mintTo: '', mintAmount: '', mintPriv: '', mintResult: '',
        xFrom: '', xTo: '', xAmount: '', xPriv: '', xResult: '',
        approver: '', spender: '', approveAmount: '', approvePriv: '', approveResult: '',
        allowanceSpender: '', allowanceResult: '',
        burnAmount: '', burnPriv: '', burnResult: '',
        infoId: '', infoResult: '',
        tfOwner: '', tfTo: '', tfAmount: '', tfSpender: '', tfSpenderKey: '', tfResult: '',
        history: [],
        async postToken(url, body) {
            const res = await apiFetch(url, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(body)
            });
            return JSON.stringify(await res.json(), null, 2);
        },
        async createToken() {
            try {
                const res = await apiFetch('/api/v1/token/create', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ name: this.name, symbol: this.symbol, total_supply: this.supply })
                });
                const data = await res.json();
                this.createResult = JSON.stringify(data, null, 2);
            } catch (e) {
                this.createResult = 'Error: ' + e.message;
            }
        },
        async getBalance() {
            try {
                const res = await apiFetch('/api/v1/token/balance?token_id=' + encodeURIComponent(this.tokenId) + '&owner=' + encodeURIComponent(this.owner));
                this.balance = JSON.stringify(await res.json(), null, 2);
            } catch (e) {
                this.balance = 'Error: ' + e.message;
            }
        },
        async info() {
            try {
                const res = await apiFetch('/api/v1/token/info?token_id=' + encodeURIComponent(this.infoId));
                this.infoResult = JSON.stringify(await res.json(), null, 2);
            } catch (e) {
                this.infoResult = 'Error: ' + e.message;
            }
        },
        async mint() {
            try {
                const res = await apiFetch('/api/v1/token/mint', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ token_id: this.tokenId, to: this.mintTo, amount: this.mintAmount, private_key: this.mintPriv })
                });
                this.mintResult = JSON.stringify(await res.json(), null, 2);
                await this.getBalance();
            } catch (e) {
                this.mintResult = 'Error: ' + e.message;
            }
        },
        async transfer() {
            try {
                this.xResult = await this.postToken('/api/v1/token/transfer', {
                    token_id: this.tokenId, from: this.xFrom, to: this.xTo,
                    amount: this.xAmount, private_key: this.xPriv
                });
            } catch (e) {
                this.xResult = 'Error: ' + e.message;
            }
        },
        async approve() {
            try {
                this.approveResult = await this.postToken('/api/v1/token/approve', {
                    token_id: this.tokenId, owner: this.approver, spender: this.spender,
                    amount: this.approveAmount, private_key: this.approvePriv
                });
            } catch (e) {
                this.approveResult = 'Error: ' + e.message;
            }
        },
        async getAllowance() {
            try {
                const res = await apiFetch('/api/v1/token/allowance?token_id=' + encodeURIComponent(this.tokenId) +
                    '&owner=' + encodeURIComponent(this.owner) + '&spender=' + encodeURIComponent(this.allowanceSpender));
                this.allowanceResult = JSON.stringify(await res.json(), null, 2);
            } catch (e) {
                this.allowanceResult = 'Error: ' + e.message;
            }
        },
        async burn() {
            try {
                // Burn has its own amount field (not the Transfer xAmount) so
                // a transfer quantity can never leak into a destroy (TASK-149).
                this.burnResult = await this.postToken('/api/v1/token/burn', {
                    token_id: this.tokenId, from: this.owner, amount: this.burnAmount,
                    private_key: this.burnPriv
                });
            } catch (e) {
                this.burnResult = 'Error: ' + e.message;
            }
        },
        async transferFrom() {
            try {
                this.tfResult = await this.postToken('/api/v1/token/transfer_from', {
                    token_id: this.tokenId, owner: this.tfOwner, to: this.tfTo,
                    amount: this.tfAmount, spender: this.tfSpender, spender_key: this.tfSpenderKey
                });
            } catch (e) {
                this.tfResult = 'Error: ' + e.message;
            }
        },
        async loadHistory() {
            try {
                const res = await apiFetch('/api/v1/token/history?token_id=' + encodeURIComponent(this.tokenId) + '&owner=' + encodeURIComponent(this.owner));
                const data = await res.json();
                this.history = Array.isArray(data) ? data : (Array.isArray(data && data.data) ? data.data : []);
            } catch (e) {
                this.history = [];
            }
        },
        async showHistory() {
            await this.loadHistory();
        }
    };
}

function oracleApp() {
    return {
        sources: [], loading: true,
        health: [], loadingHealth: true,
        addName: '', addUrl: '', addType: '', addMethod: '', addPath: '', addInterval: 60, addResult: '',
        templates: [],
        fetchSource: '', fetchResult: '',
        querySource: '', queryLimit: 10, queryRows: [], queryError: '',
        sourcesError: '',
        latestSource: '', latestResult: '',
        async init() {
            await this.refresh();
            // The Oracle scheduler updates sources/health in the background, so
            // re-poll so the list and feed health stay current (ISS-125).
            startPolling(this, 15000);
        },
        // refresh re-snapshots sources + feed health + templates; exposed as
        // the header "↻ Refresh" button and driven by startPolling.
        async refresh() {
            await Promise.all([this.listSources(), this.loadHealth(), this.loadTemplates()]);
        },
        async loadTemplates() {
            try {
                const res = await apiFetch('/api/v1/oracle/templates');
                const data = await res.json();
                this.templates = Array.isArray(data) ? data : [];
            } catch (e) { this.templates = []; }
        },
        applyTemplate(id) {
            const t = this.templates.find(x => x.id === id);
            if (!t) return;
            this.addName = t.name || '';
            this.addUrl = t.url || '';
            this.addType = t.type || '';
            this.addMethod = t.method || '';
            this.addPath = t.path || '';
            this.addInterval = t.interval || 60;
        },
        async loadHealth() {
            this.loadingHealth = true;
            try {
                const res = await apiFetch('/api/v1/oracle/health');
                const data = await res.json();
                this.health = Array.isArray(data) ? data : [];
            } catch (e) { this.health = []; }
            this.loadingHealth = false;
        },
        async addSource() {
            try {
                const res = await apiFetch('/api/v1/oracle/sources', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        name: this.addName,
                        url: this.addUrl,
                        type: this.addType || undefined,
                        method: this.addMethod || undefined,
                        path: this.addPath || undefined,
                        interval: parseInt(this.addInterval)
                    })
                });
                const data = await res.json();
                if (!res.ok) { this.addResult = 'Error: ' + (data.error || data.message || res.status); return; }
                this.addResult = 'Source "' + data.name + '" added (id: ' + data.id + ')';
                this.addName = ''; this.addUrl = ''; this.addType = ''; this.addMethod = ''; this.addPath = '';
                await this.listSources();
            } catch (e) {
                this.addResult = 'Error: ' + e.message;
            }
        },
        async setEnabled(source, enabled) {
            this.sourcesError = '';
            try {
                await apiFetch('/api/v1/oracle/sources/' + encodeURIComponent(source.id), {
                    method: 'PATCH',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ enabled: enabled })
                });
                await this.listSources();
            } catch (e) { this.sourcesError = 'Error: ' + e.message; }
        },
        async deleteSource(source) {
            if (!confirm('Delete source "' + (source.name || source.id) + '"?')) return;
            this.sourcesError = '';
            try {
                // The destructive-op guard (delete confirm) mirrors the CLI
                // --confirm gate; apiFetch raises the shared banner on failure.
                await apiFetch('/api/v1/oracle/sources/' + encodeURIComponent(source.id), {
                    method: 'DELETE'
                });
                await this.listSources();
            } catch (e) { this.sourcesError = 'Error: ' + e.message; }
        },
        async listSources() {
            this.loading = true;
            try {
                const res = await apiFetch('/api/v1/oracle/sources');
                const data = await res.json();
                this.sources = (data && data.sources) || [];
            } catch (e) { this.sources = []; }
            this.loading = false;
        },
        async fetch() {
            try {
                const res = await apiFetch('/api/v1/oracle/fetch', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ source: this.fetchSource })
                });
                this.fetchResult = JSON.stringify(await res.json(), null, 2);
            } catch (e) {
                this.fetchResult = 'Error: ' + e.message;
            }
        },
        async query() {
            this.queryError = '';
            try {
                const res = await apiFetch('/api/v1/oracle/query?source=' + encodeURIComponent(this.querySource) + '&limit=' + this.queryLimit);
                const data = await res.json();
                this.queryRows = (data && data.data) || [];
            } catch (e) {
                this.queryRows = [];
                this.queryError = 'Error: ' + e.message;
            }
        },
        async latest() {
            try {
                const res = await apiFetch('/api/v1/oracle/latest?source=' + encodeURIComponent(this.latestSource));
                this.latestResult = JSON.stringify(await res.json(), null, 2);
            } catch (e) {
                this.latestResult = 'Error: ' + e.message;
            }
        }
    };
}
