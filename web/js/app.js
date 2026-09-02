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
// apiFetch aborts requests that stall longer than this (ISS-185). The API
// server bounds its own writes at 30s, so 20s here is comfortably shorter —
// only genuinely stuck connections trip it.
const REQUEST_TIMEOUT_MS = 20000;

async function apiFetch(path, options) {
    // Merge the caller's headers (e.g. Content-Type for JSON bodies) ON TOP of
    // the API key rather than in place of it. A naive
    // Object.assign({headers: auroraHeaders()}, options) lets an options.headers
    // key REPLACE the whole headers object, silently dropping X-API-Key and
    // turning every web write into a 401 (round-97 apiFetch refactor regression,
    // ISS-160). GET/DELETE calls pass no options.headers and were unaffected.
    const headers = Object.assign(auroraHeaders(), (options && options.headers) || {});
    // Abort a stalled request after 20s (ISS-185): a connection that never
    // responds otherwise leaves every withBusy busy.<name> flag set forever
    // (all submit buttons :disabled) and startPolling's running flag stuck,
    // both until a full page reload. The abort surfaces as a normal fetch
    // rejection, so withBusy's settled() clears the flag and the poll can
    // resume.
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);
    let res;
    try {
        res = await fetch(path, Object.assign({}, options, { headers, signal: controller.signal }));
    } catch (e) {
        if (e && e.name === 'AbortError') {
            const msg = 'Request timed out after ' + REQUEST_TIMEOUT_MS / 1000 + 's';
            window.showApiError(msg);
            throw new Error(msg);
        }
        throw e;
    } finally {
        clearTimeout(timer);
    }
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

// withBusy wraps the named async actions of an Alpine app object so each sets
// a matching `busy.<name>` flag while its request is in flight. Submit buttons
// bind `:disabled="busy.<name>"`, turning a double-click on a create/transfer
// form into a no-op instead of a duplicate record (TASK-177, ISS-175).
//
// The blockchain page already guarded its single verify button inline with
// :disabled="loading"; this generalizes exactly that guard to every write
// form while keeping the flag per-action, so one busy create never disables an
// unrelated form on the same page. Writes go through `this` (the Alpine
// reactive component) rather than the raw object so :disabled re-renders —
// mutating the captured raw object would bypass Alpine's proxy and never
// update the DOM.
function withBusy(app, actions) {
    app.busy = {};
    for (const name of actions) {
        app.busy[name] = false;
        const fn = app[name];
        app[name] = function (...args) {
            if (this.busy[name]) return Promise.resolve();
            this.busy[name] = true;
            const settled = () => { this.busy[name] = false; };
            return Promise.resolve().then(() => fn.apply(this, args)).then(settled, settled);
        };
    }
    return app;
}

function lotteryApp() {
    return withBusy({
        participants: '',
        seed: '',
        count: 1,
        result: '',
        history: [],
        historyFailed: false,
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
                this.historyFailed = false;
            } catch (e) {
                // Distinguish "no lotteries yet" from "couldn't load" so a
                // first-load failure isn't misread as an empty system — same
                // ISS-191 contract TASK-195 applied to voting/oracle. And
                // apply the keep-prior-rows policy (TASK-151): a TRANSIENT
                // refresh failure (e.g. createLottery's follow-up reload
                // times out) must NOT blank draws that already rendered —
                // history is left untouched so a known-good list stays on
                // screen while the API recovers; historyFailed flags the
                // failure either way (TASK-250, ISS-246).
                this.historyFailed = true;
            }
            this.loading = false;
        },
        async createLottery() {
            // Validate client-side against the API contract (seed ≥3 chars,
            // whole winner count in [1, participant count]) so a mismatch is
            // explained instead of surfacing as an opaque 400.
            const participants = this.participants.split(',').map(p => p.trim()).filter(Boolean);
            const seed = (this.seed || '').trim();
            const count = Number(this.count);
            if (seed.length < 3) {
                this.result = 'Seed is required and must be at least 3 characters.';
                return;
            }
            if (!Number.isInteger(count) || count < 1) {
                this.result = 'Winner count must be a whole number of at least 1.';
                return;
            }
            if (count > participants.length) {
                this.result = 'Winner count (' + count + ') cannot exceed the number of participants (' + participants.length + ').';
                return;
            }
            try {
                const res = await apiFetch('/api/v1/lottery/create', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        participants: participants.join(','),
                        seed: seed,
                        winner_count: count
                    })
                });
                const data = await res.json();
                this.result = JSON.stringify(data, null, 2);
                await this.loadHistory();
                this.participants = '';
                this.seed = '';
                this.count = 1;
                // Advance the Verify Draw id to the fresh draw so the operator
                // can verify the provably-fair result without copying the id
                // from the JSON result block (mirrors TASK-150/ISS-148).
                if (data && data.id) this.verifyId = data.id;
            } catch (e) {
                this.result = 'Error: ' + e.message;
            }
        }
    }, ['createLottery', 'verifyDraw']);
}

function dashboardApp() {
    return {
        stats: { lotteries: 0, votes: 0, candidates: 0, sessions: 0, integrity: '-', oracle: '-' },
        activity: [],
        // loaded flips true the moment any stat/activity endpoint answers, so
        // a total first-load failure is never presented as a genuinely empty
        // system (0/0/0/0 + "No recent activity").
        loaded: false,
        refreshing: false,
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
        // Reentrancy is guarded so rapid manual clicks can't stack overlapping
        // refreshes alongside the 15s timer.
        async refresh() {
            if (this.refreshing) return;
            this.refreshing = true;
            try {
                const activity = await Promise.all([
                    this.loadLotteries(),
                    this.loadVoting(),
                    this.loadBlockchain(),
                    this.loadOracleHealth()
                ]);
                // A list loader returns null on a failed cycle (vs [] for a
                // success with no rows). When EVERY source of activity rows
                // fails on a transient poll blip, keep the already-rendered
                // list instead of collapsing it into a false "No recent
                // activity" empty-state — the same keep-prior-rows policy the
                // stat cards already follow (TASK-151). The cards' loaders
                // (blockchain/oracle-health) never add rows, so they don't
                // participate in this decision.
                const lists = activity.slice(0, 2).filter(r => r !== null);
                if (lists.length > 0) {
                    this.activity = [].concat(...lists);
                }
            } finally {
                this.refreshing = false;
            }
        },
        async loadLotteries() {
            const entries = [];
            try {
                const res = await apiFetch('/api/v1/lottery/history');
                const data = await res.json();
                if (Array.isArray(data)) {
                    this.stats.lotteries = data.length;
                    this.loaded = true;
                    data.slice(0, 10).forEach(l => {
                        entries.push({
                            key: 'lot-' + (l.id || Math.random()),
                            title: 'Lottery ' + (l.id || ''),
                            detail: (l.winners || []).length + ' winner(s)'
                        });
                    });
                }
            } catch (e) {
                // A never-loaded card marks itself unavailable ('?') instead of
                // reporting a guessed 0; once a real value has been seen it is
                // kept across transient poll failures (keep-prior-rows policy).
                if (!this.loaded) this.stats.lotteries = '?';
                console.error(e);
                return null; // failed cycle: signal refresh to keep prior rows
            }
            return entries;
        },
        async loadVoting() {
            // Each stat group has its own try/catch so a transient failure on
            // one endpoint (e.g. sessions 500) blanks only its own card instead
            // of the shared Promise.all dropping the sibling stats too (TASK-147).
            const groups = await Promise.all([this.loadCandidatesStats(), this.loadSessionsStats()]);
            // Candidates never contribute activity rows; sessions do. Keep the
            // null (failed-cycle) signal from the only voting source of rows so
            // refresh() can preserve the rendered list on an all-endpoint blip
            // (TASK-258, ISS-254; loadCandidatesStats [] is a success-with-no-rows).
            return groups[1] === null ? null : [].concat(...groups);
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
                    this.loaded = true;
                }
            } catch (e) {
                if (!this.loaded) { this.stats.candidates = '?'; this.stats.votes = '?'; }
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
                    this.loaded = true;
                    sessions.slice(0, 10).forEach(s => {
                        entries.push({
                            key: 'sess-' + s.id,
                            title: 'Session: ' + s.title,
                            detail: 'Status: ' + s.status + ' · ' + (s.candidates || []).length + ' candidate(s)'
                        });
                    });
                }
            } catch (e) {
                if (!this.loaded) this.stats.sessions = '?';
                console.error(e);
                return null; // failed cycle: signal refresh to keep prior rows
            }
            return entries;
        },
        async loadBlockchain() {
            try {
                const report = await (await apiFetch('/api/v1/blockchain/verify')).json();
                this.stats.integrity = report.valid ? 'OK' : 'BROKEN';
            } catch (e) {
                // Keep a previously-seen value across transient poll failures,
                // like every sibling loader (loadLotteries/loadCandidatesStats/
                // loadSessionsStats): only a card that never loaded marks itself
                // '?' (TASK-234, ISS-232; TASK-151 keep-prior policy).
                if (!this.loaded) this.stats.integrity = '?';
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
                if (!this.loaded) this.stats.oracle = '?';
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
            // Drop the previous report up front so a failed verify can never
            // leave a stale green "Valid: Yes" rendered beneath the new error
            // line (contradictory output, ISS-187).
            this.report = null;
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
    return withBusy({
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
                // the JSON result block (TASK-150). Every form (History
                // included) keys off this single id, so one advance fills them
                // all — a separate historyId would silently desync on manual
                // edits (ISS-255, mirroring the token page's shared tokenId).
                if (data && data.id) this.id = data.id;
                if (data && data.owner) this.owner = data.owner;
                // The Transfer form keys off its own from field; advance it to
                // the fresh owner so the post-mint transfer needs no manual
                // copy of the creator key (same TASK-150 advance contract).
                if (data && data.owner) this.from = data.owner;
                // Clear the mint form once the mint committed (the result JSON
                // below keeps the committed values; consistent with the
                // voting/oracle forms clearing on success).
                this.name = ''; this.description = ''; this.imageUrl = ''; this.tokenUri = ''; this.creator = '';
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
                // History keys off the SAME shared id as Get/Transfer/Burn — no
                // separate historyId field. The token page made every form share
                // one tokenId context; NFT followed the multi-field pattern, and
                // a manually edited shared id silently desynced the History box
                // to a stale value, showing the wrong NFT's history (ISS-255).
                const res = await apiFetch('/api/v1/nft/' + encodeURIComponent(this.id) + '/history');
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
                // Advance the shared owner context to the recipient, mirroring
                // the mint/ token-mint context advances (TASK-150/151): the
                // List-by-Owner and Burn forms otherwise still target the
                // pre-transfer owner and Burn errors "not the owner" (ISS-186).
                if (this.to) this.owner = this.to;
            } catch (e) {
                this.transferResult = 'Error: ' + e.message;
            }
        },
        async burn() {
            // Destroying an NFT is permanent and irreversible — one mis-click
            // (or Enter on the submit while the other fields auto-filled from
            // mint/transfer/get) would destroy an asset with no undo. The
            // destructive-op guard mirrors the CLI --confirm gate and the
            // oracle Delete confirm (ISS-253).
            if (!confirm('Burn NFT "' + (this.id || '') + '" permanently?')) return;
            this.burnResult = '';
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
    }, ['mint', 'list', 'get', 'history', 'transfer', 'burn']);
}

function votingApp() {
    return withBusy({
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
        candidatesFailed: false,
        sessions: [],
        sessionsFailed: false,
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
                this.candidatesFailed = false;
            } catch (e) {
                // Distinguish "no candidates created yet" from "couldn't load":
                // a load failure must not masquerade as an empty roster that
                // blocks create-session and cast-vote (ISS-191 parity with
                // sessions).
                this.candidates = [];
                this.candidatesFailed = true;
                console.error(e);
            }
        },
        // The voting page has no polling, so a transient failure at init leaves
        // candidatesFailed/sessionsFailed until a full reload — which disables
        // Cast Vote permanently in-page. The Retry button re-runs the loaders
        // instead (TASK-243, ISS-241).
        async retryLoad() {
            this.loading = true;
            await Promise.all([this.loadCandidates(), this.loadSessions()]);
            this.loading = false;
        },
        async loadSessions() {
            try {
                const res = await apiFetch('/api/v1/voting/sessions');
                const data = await res.json();
                this.sessions = Array.isArray(data) ? data : [];
                this.sessionsFailed = false;
            } catch (e) {
                // Distinguish "nothing created yet" from "couldn't load" so a
                // first-load failure (no polling on this page; the banner shows
                // the API error) isn't misread as an empty system (ISS-191).
                this.sessions = [];
                this.sessionsFailed = true;
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
                // Advance the shared session-id fields to the freshly created
                // session (mirrors the NFT mint context advance, TASK-150): the
                // operator's natural next steps — start/end, cast a vote, view
                // results — all key off a session id, so no manual copy from the
                // result line is needed (ISS-148).
                if (data && data.id) {
                    this.voteSessionId = data.id;
                    this.controlSessionId = data.id;
                    this.resultsSessionId = data.id;
                }
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
        // The Session ID input is bound to @keydown.enter.prevent so pressing
        // Enter after typing an ID does NOT implicitly submit the form — the
        // submit handler is startSession, and an operator finishing a session
        // control line could otherwise silently reactivate an ended election
        // (the backend deliberately permits end->active reopen per DEC-004, but
        // it must be an explicit click on the button, not a typing accident;
        // TASK-236, ISS-234). The .prevent modifier does the work; the empty
        // handler keeps the directive's expression self-documenting.
        holdEnterKey() {},
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
    }, ['registerVoter', 'registerCandidate', 'createSession', 'castVote',
        'loadResults', 'startSession', 'endSession']);
}

function tokenApp() {
    return withBusy({
        name: '', symbol: '', supply: '', decimals: 8, createOwner: '', createResult: '',
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
            // The decimals input is type=number min=0 max=127, but min/max only
            // constrain the spinner, not keyboard typing — a typed 200 was sent
            // verbatim and Go's encoding/json rejected it as an out-of-range
            // int8 ("invalid request (400)") with no hint of the real bound.
            // Clamp and reflect the value back so the visible field always
            // matches the request (same contract as oracle clampLimit, ISS-184;
            // domain accepts 0..127, ISS-255).
            const parsedDecimals = (this.decimals === undefined || this.decimals === '') ? undefined : parseInt(this.decimals, 10);
            if (parsedDecimals !== undefined && !isNaN(parsedDecimals)) {
                this.decimals = Math.min(127, Math.max(0, parsedDecimals));
            }
            try {
                const res = await apiFetch('/api/v1/token/create', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        name: this.name,
                        symbol: this.symbol,
                        total_supply: this.supply,
                        // The API's decimals is a pointer: an explicit 0 is a
                        // real 0-decimals token, an omitted field defaults to
                        // 8 in the domain. Send undefined when the operator
                        // clears the field so the two stay distinct; matching
                        // the CLI's --decimals range (0..MaxInt8) (ISS-195).
                        // The field was clamped to 0..127 above so this is
                        // always a valid int8 (ISS-255).
                        decimals: (this.decimals === undefined || this.decimals === '') ? undefined : parseInt(this.decimals, 10),
                        // The API requires a valid owner public key; without it
                        // every create 400s with PUBLIC_KEY_REQUIRED (ISS-147).
                        // The key comes from the create form's own createOwner
                        // field — never the shared inspection owner — so a key
                        // typed for a create cannot leak into Balance/History
                        // context (isolation like TASK-149).
                        owner: this.createOwner
                    })
                });
                const data = await res.json();
                this.createResult = JSON.stringify(data, null, 2);
                // Advance the shared Token ID / Owner fields to the freshly
                // created token (mirrors the NFT mint context advance, TASK-150):
                // the Balance/Info/Mint/Transfer/Approve/Allowance/Burn/History
                // forms all key off these two, so the next step needs no manual
                // copy from the JSON result block (ISS-148). Advancing is
                // deliberate: the created token's owner is now the context the
                // operator wants to inspect.
                if (data && data.id) this.tokenId = data.id;
                if (data && data.owner) this.owner = data.owner;
                // The Token Info form keys off its own infoId field, so the
                // create advance must populate it too — otherwise Info stays
                // blank right after a create while Balance/History auto-fill
                // (the same TASK-150 context-advance contract, ISS-195).
                if (data && data.id) this.infoId = data.id;
                // Clear the create form once the token committed (result JSON
                // keeps the values; consistent clearing on success).
                this.name = ''; this.symbol = ''; this.supply = ''; this.decimals = 8; this.createOwner = '';
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
                const data = await res.json();
                this.mintResult = JSON.stringify(data, null, 2);
                // A successful mint credits the recipient; advance the shared
                // owner + tokenId to that context so the auto-refreshed balance
                // below queries the freshly credited account instead of the
                // (possibly blank) shared owner — which previously surfaced a
                // confusing "token_id and owner required" error right after a
                // success (ISS-150, mirrors TASK-150).
                if (data && data.to) this.owner = data.to;
                if (data && data.token_id) this.tokenId = data.token_id;
                // The Transfer form keys off its own from field (xFrom);
                // advance it to the mint recipient so the post-mint transfer
                // needs no manual key copy (same TASK-150 advance contract).
                if (data && data.to) this.xFrom = data.to;
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
                // A successful transfer credits the recipient; advance the
                // shared owner (Balance/History/Allowance/Burn) and the
                // Transfer form's From field to that context, then refresh the
                // balance — mint's TASK-150 advance contract. Without this the
                // next Get Balance showed the drained sender and read as "the
                // transfer failed" (TASK-235, ISS-233).
                const data = JSON.parse(this.xResult);
                if (data && data.to) { this.owner = data.to; this.xFrom = data.to; }
                await this.getBalance();
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
            // Burning tokens is permanent and irreversible — a mis-click or a
            // stray Enter destroys value with no undo. The destructive-op
            // guard mirrors the CLI --confirm gate and the NFT/oracle Deletes
            // (ISS-253).
            if (!confirm('Burn ' + (this.burnAmount || '') + ' of token "' + (this.tokenId || '') + '" permanently?')) return;
            this.burnResult = '';
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
                // The tokens now sit with the recipient; advance the shared
                // owner so the inspection forms read the freshly-funded account
                // (mint's TASK-150 advance contract). tfOwner is left alone — it
                // is the payer whose allowance was spent (TASK-235, ISS-233).
                const data = JSON.parse(this.tfResult);
                if (data && data.to) this.owner = data.to;
                await this.getBalance();
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
    }, ['createToken', 'getBalance', 'info', 'mint', 'transfer', 'approve',
        'getAllowance', 'burn', 'transferFrom', 'showHistory']);
}

function oracleApp() {
    return withBusy({
        sources: [], sourcesFailed: false, loading: true,
        health: [], loadingHealth: true, healthFailed: false,
        refreshing: false,
        addName: '', addUrl: '', addType: '', addMethod: '', addPath: '', addInterval: 60, addResult: '',
        templates: [],
        fetchSource: '', fetchResult: '',
        querySource: '', queryLimit: 10, queryRows: [], queryError: '', queried: false,
        sourcesError: '',
        latestSource: '', latestResult: '',
        async init() {
            await this.refresh();
            // The Oracle scheduler updates sources/health in the background, so
            // re-poll so the list and feed health stay current (ISS-125).
            startPolling(this, 15000);
        },
        // refresh re-snapshots sources + feed health + templates; exposed as
        // the header "↻ Refresh" button and driven by startPolling. Reentrancy
        // is guarded so rapid manual clicks can't stack overlapping refreshes
        // (the timer path and the button share this one entry point).
        async refresh() {
            if (this.refreshing) return;
            this.refreshing = true;
            try {
                await Promise.all([this.listSources(), this.loadHealth(), this.loadTemplates()]);
            } finally {
                this.refreshing = false;
            }
        },
        sourceName(id) {
            const s = this.sources.find(x => x.id === id);
            return s ? s.name : id;
        },
        async loadTemplates() {
            try {
                const res = await apiFetch('/api/v1/oracle/templates');
                const data = await res.json();
                this.templates = Array.isArray(data) ? data : [];
            } catch (e) { /* keep prior rows; apiFetch's shared banner shows the error */ }
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
                this.healthFailed = false;
            } catch (e) {
                // Keep prior rows during a background poll refresh, but flag a
                // first-load failure so "(no feed activity yet)" isn't shown
                // for an API error (same ISS-191 contract as sources above).
                this.healthFailed = true;
            }
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
                // apiFetch already throws on non-2xx and surfaces the API error
                // in the shared banner, so res is guaranteed ok here; the catch
                // below handles failures (dead guard removed, round-107's six
                // votingApp guards were the same class — ISS-149).
                const data = await res.json();
                this.addResult = 'Source "' + data.name + '" added (id: ' + data.id + ')';
                this.addName = ''; this.addUrl = ''; this.addType = ''; this.addMethod = ''; this.addPath = '';
                // Advance the shared Fetch/Query/Latest source ids to the fresh
                // source (mirrors TASK-150): a newly added source's natural next
                // step is fetching/querying it, and that needed a manual id copy
                // from the result line before (ISS-151).
                if (data && data.id) {
                    this.fetchSource = data.id;
                    this.querySource = data.id;
                    this.latestSource = data.id;
                }
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
                this.sourcesFailed = false;
            } catch (e) {
                // Keep prior rows during a background poll refresh, but flag a
                // first-load failure so "(no sources)" isn't shown for an API
                // error the banner is already surfacing (ISS-191).
                this.sourcesFailed = true;
            }
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
        // clampLimit mirrors the CLI's clampQueryLimit and the TUI's
        // clampQueryLimitValue (TASK-178): the query limit is bounded to
        // [1,100] client-side so a typed 0 or 500 is never sent raw to the
        // server to be silently coerced without feedback (ISS-184). The
        // number input's min/max attributes only constrain the spinner, not
        // keyboard typing.
        clampLimit(raw) {
            const v = parseInt(raw, 10);
            if (isNaN(v) || v <= 0) return 10;
            return Math.min(v, 100);
        },
        async query() {
            this.queryError = '';
            this.queried = true;
            try {
                const limit = this.clampLimit(this.queryLimit);
                // Reflect the clamped value back so the visible input always
                // matches what is actually sent (a typed 500 -> shows 100;
                // without this the input lied while the request was clamped,
                // ISS-184 feedback contract).
                this.queryLimit = limit;
                const res = await apiFetch('/api/v1/oracle/query?source=' + encodeURIComponent(this.querySource) + '&limit=' + limit);
                const data = await res.json();
                // [] vs [0 rows] distinction: a successful-but-empty query must
                // be distinguishable from "never queried" — the query table
                // shows only for non-empty rows, so a zero-row success otherwise
                // looks like a no-op and silently clears prior rows (ISS-255).
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
    }, ['addSource', 'setEnabled', 'deleteSource', 'fetch', 'query', 'latest']);
}
