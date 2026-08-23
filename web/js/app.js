// auroraHeaders returns fetch headers that authenticate the same-origin API
// calls. The API gateway injects the resolved key into served HTML as
// window.AURORA_API_KEY; without it every /api/v1 call would return 401.
function auroraHeaders(extra) {
    return Object.assign({ 'X-API-Key': window.AURORA_API_KEY || '' }, extra || {});
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
                const res = await fetch('/api/v1/lottery/' + encodeURIComponent(this.verifyId) + '/verify', { headers: auroraHeaders() });
                this.verifyResult = JSON.stringify(await res.json(), null, 2);
            } catch (e) {
                this.verifyResult = 'Error: ' + e.message;
            }
        },
        async loadHistory() {
            this.loading = true;
            try {
                const res = await fetch('/api/v1/lottery/history', { headers: auroraHeaders() });
                this.history = await res.json();
            } catch (e) {
                console.error(e);
            }
            this.loading = false;
        },
        async createLottery() {
            try {
                const res = await fetch('/api/v1/lottery/create', {
                    method: 'POST',
                    headers: auroraHeaders({ 'Content-Type': 'application/json' }),
                    body: JSON.stringify({
                        participants: this.participants.split(',').map(p => p.trim()),
                        seed: this.seed || undefined,
                        count: parseInt(this.count)
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
            await Promise.all([
                this.loadLotteries(),
                this.loadVoting(),
                this.loadBlockchain(),
                this.loadOracleHealth()
            ]);
            this.loading = false;
        },
        async loadLotteries() {
            try {
                const res = await fetch('/api/v1/lottery/history', { headers: auroraHeaders() });
                const data = await res.json();
                if (Array.isArray(data)) {
                    this.stats.lotteries = data.length;
                    data.slice(0, 10).forEach(l => {
                        this.activity.push({
                            key: 'lot-' + (l.id || Math.random()),
                            title: 'Lottery ' + (l.id || ''),
                            detail: (l.winners || []).length + ' winner(s)'
                        });
                    });
                }
            } catch (e) {
                console.error(e);
            }
        },
        async loadVoting() {
            try {
                const [candRes, sessRes] = await Promise.all([
                    fetch('/api/v1/voting/candidates', { headers: auroraHeaders() }),
                    fetch('/api/v1/voting/sessions', { headers: auroraHeaders() })
                ]);
                const candidates = await candRes.json();
                const sessions = await sessRes.json();
                if (Array.isArray(candidates)) {
                    this.stats.candidates = candidates.length;
                    // "Total votes" has no dedicated endpoint; derive it from
                    // each candidate's real vote_count.
                    this.stats.votes = candidates.reduce((sum, c) => sum + (c.vote_count || 0), 0);
                }
                if (Array.isArray(sessions)) {
                    this.stats.sessions = sessions.length;
                    sessions.slice(0, 10).forEach(s => {
                        this.activity.push({
                            key: 'sess-' + s.id,
                            title: 'Session: ' + s.title,
                            detail: 'Status: ' + s.status + ' · ' + (s.candidates || []).length + ' candidate(s)'
                        });
                    });
                }
            } catch (e) {
                console.error(e);
            }
        },
        async loadBlockchain() {
            try {
                const res = await fetch('/api/v1/blockchain/verify', { headers: auroraHeaders() });
                if (!res.ok) {
                    this.stats.integrity = '?';
                    return;
                }
                const report = await res.json();
                this.stats.integrity = report.valid ? 'OK' : 'BROKEN';
            } catch (e) {
                this.stats.integrity = '?';
                console.error(e);
            }
        },
        async loadOracleHealth() {
            try {
                const res = await fetch('/api/v1/oracle/health', { headers: auroraHeaders() });
                if (!res.ok) { this.stats.oracle = '?'; return; }
                const feeds = await res.json();
                if (!Array.isArray(feeds) || feeds.length === 0) { this.stats.oracle = '-'; return; }
                const healthy = feeds.filter(f => f.successes > 0 && f.failures === 0).length;
                const failed = feeds.filter(f => f.failures > 0).length;
                this.stats.oracle = healthy + ' OK' + (failed ? ' · ' + failed + ' fail' : '');
            } catch (e) {
                this.stats.oracle = '?';
                console.error(e);
            }
        }
    };
}

// blockchainApp drives the /blockchain.html ledger integrity page (v1.26). It
// calls the protected GET /api/v1/blockchain/verify endpoint and renders the
// IntegrityReport (valid, chain length, first broken index + reason).
function blockchainApp() {
    return {
        report: null,
        error: '',
        loading: false,
        async verify() {
            this.loading = true;
            this.error = '';
            try {
                const res = await fetch('/api/v1/blockchain/verify', { headers: auroraHeaders() });
                if (!res.ok) {
                    this.error = 'Verification failed: HTTP ' + res.status;
                    return;
                }
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
            this.mintResult = await this.post('/api/v1/nft/mint', {
                name: this.name,
                description: this.description || undefined,
                image_url: this.imageUrl || undefined,
                token_uri: this.tokenUri || undefined,
                creator: this.creator
            });
        },
        async list() {
            const res = await fetch('/api/v1/nft/list?owner=' + encodeURIComponent(this.owner), { headers: auroraHeaders() });
            this.listResult = await this.text(res);
        },
        async get() {
            const res = await fetch('/api/v1/nft/' + encodeURIComponent(this.id), { headers: auroraHeaders() });
            this.getResult = await this.text(res);
        },
        async history() {
            try {
                const res = await fetch('/api/v1/nft/' + encodeURIComponent(this.historyId) + '/history', { headers: auroraHeaders() });
                this.historyResult = JSON.stringify(await res.json(), null, 2);
            } catch (e) {
                this.historyResult = 'Error: ' + e.message;
            }
        },
        async transfer() {
            this.transferResult = await this.post('/api/v1/nft/transfer', {
                nft_id: this.id,
                from: this.from,
                to: this.to,
                private_key: this.privateKey
            });
        },
        async burn() {
            this.burnResult = await this.post('/api/v1/nft/burn', {
                nft_id: this.id,
                owner: this.owner,
                private_key: this.privateKey
            });
        },
        async post(url, body) {
            const res = await fetch(url, {
                method: 'POST',
                headers: auroraHeaders({ 'Content-Type': 'application/json' }),
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
                const res = await fetch('/api/v1/voting/candidates', { headers: auroraHeaders() });
                this.candidates = await res.json();
            } catch (e) {
                console.error(e);
            }
        },
        async loadSessions() {
            try {
                const res = await fetch('/api/v1/voting/sessions', { headers: auroraHeaders() });
                this.sessions = await res.json();
            } catch (e) {
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
                const res = await fetch('/api/v1/voting/register/voter', {
                    method: 'POST',
                    headers: auroraHeaders({ 'Content-Type': 'application/json' }),
                    body: JSON.stringify({ name: this.voterName })
                });
                const data = await res.json();
                if (!res.ok) { this.voterResult = 'Error: ' + (data.message || res.status); return; }
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
                const res = await fetch('/api/v1/voting/register/candidate', {
                    method: 'POST',
                    headers: auroraHeaders({ 'Content-Type': 'application/json' }),
                    body: JSON.stringify({
                        name: this.candName,
                        party: this.candParty,
                        program: this.candProgram
                    })
                });
                const data = await res.json();
                if (!res.ok) { this.candResult = 'Error: ' + (data.message || res.status); return; }
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
                const res = await fetch('/api/v1/voting/session', {
                    method: 'POST',
                    headers: auroraHeaders({ 'Content-Type': 'application/json' }),
                    body: JSON.stringify({
                        title: this.sessionTitle,
                        description: this.sessionDesc,
                        candidate_ids: this.sessionCandidateIds,
                        start_time: parseInt(this.sessionStart),
                        end_time: parseInt(this.sessionEnd)
                    })
                });
                const data = await res.json();
                if (!res.ok) { this.sessionResult = 'Error: ' + (data.message || res.status); return; }
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
                const res = await fetch('/api/v1/voting/vote', {
                    method: 'POST',
                    headers: auroraHeaders({ 'Content-Type': 'application/json' }),
                    body: JSON.stringify({
                        voter_public_key: this.voteVoterPub,
                        candidate_id: this.voteCandidateId,
                        private_key: this.votePriv,
                        session_id: this.voteSessionId
                    })
                });
                const data = await res.json();
                if (!res.ok) { this.voteResult = 'Error: ' + (data.message || res.status); return; }
                this.voteResult = 'Vote recorded (id: ' + data.id + ', block height: ' + data.block_height + ')';
                await this.loadCandidates();
            } catch (e) {
                this.voteResult = 'Error: ' + e.message;
            }
        },
        async loadResults() {
            try {
                const res = await fetch('/api/v1/voting/results/' + encodeURIComponent(this.resultsSessionId), { headers: auroraHeaders() });
                const data = await res.json();
                if (!res.ok) {
                    this.results = null;
                    this.resultsError = 'Error: ' + (data.message || res.status);
                    return;
                }
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
                const res = await fetch(url, { method: 'POST', headers: auroraHeaders() });
                const data = await res.json();
                if (!res.ok) { this.controlResult = 'Error: ' + (data.message || res.status); return; }
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
        burnPriv: '', burnResult: '',
        infoId: '', infoResult: '',
        tfOwner: '', tfTo: '', tfAmount: '', tfSpender: '', tfSpenderKey: '', tfResult: '',
        history: [],
        async postToken(url, body) {
            const res = await fetch(url, {
                method: 'POST',
                headers: auroraHeaders({ 'Content-Type': 'application/json' }),
                body: JSON.stringify(body)
            });
            return JSON.stringify(await res.json(), null, 2);
        },
        async createToken() {
            try {
                const res = await fetch('/api/v1/token/create', {
                    method: 'POST',
                    headers: auroraHeaders({ 'Content-Type': 'application/json' }),
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
                const res = await fetch('/api/v1/token/balance?token_id=' + encodeURIComponent(this.tokenId) + '&owner=' + encodeURIComponent(this.owner), { headers: auroraHeaders() });
                this.balance = JSON.stringify(await res.json(), null, 2);
            } catch (e) {
                this.balance = 'Error: ' + e.message;
            }
        },
        async info() {
            try {
                const res = await fetch('/api/v1/token/info?token_id=' + encodeURIComponent(this.infoId), { headers: auroraHeaders() });
                this.infoResult = JSON.stringify(await res.json(), null, 2);
            } catch (e) {
                this.infoResult = 'Error: ' + e.message;
            }
        },
        async mint() {
            try {
                const res = await fetch('/api/v1/token/mint', {
                    method: 'POST',
                    headers: auroraHeaders({ 'Content-Type': 'application/json' }),
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
                const res = await fetch('/api/v1/token/allowance?token_id=' + encodeURIComponent(this.tokenId) +
                    '&owner=' + encodeURIComponent(this.owner) + '&spender=' + encodeURIComponent(this.allowanceSpender),
                    { headers: auroraHeaders() });
                this.allowanceResult = JSON.stringify(await res.json(), null, 2);
            } catch (e) {
                this.allowanceResult = 'Error: ' + e.message;
            }
        },
        async burn() {
            try {
                this.burnResult = await this.postToken('/api/v1/token/burn', {
                    token_id: this.tokenId, from: this.owner, amount: this.xAmount,
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
                const res = await fetch('/api/v1/token/history?token_id=' + encodeURIComponent(this.tokenId) + '&owner=' + encodeURIComponent(this.owner), { headers: auroraHeaders() });
                const data = await res.json();
                this.history = Array.isArray(data) ? data : (data.data || []);
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
        fetchSource: '', fetchResult: '',
        querySource: '', queryLimit: 10, queryRows: [],
        async init() { await Promise.all([this.listSources(), this.loadHealth()]); },
        async loadHealth() {
            this.loadingHealth = true;
            try {
                const res = await fetch('/api/v1/oracle/health', { headers: auroraHeaders() });
                const data = await res.json();
                this.health = Array.isArray(data) ? data : [];
            } catch (e) { this.health = []; }
            this.loadingHealth = false;
        },
        async addSource() {
            try {
                const res = await fetch('/api/v1/oracle/sources', {
                    method: 'POST',
                    headers: auroraHeaders({ 'Content-Type': 'application/json' }),
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
                if (!res.ok) { this.addResult = 'Error: ' + (data.message || res.status); return; }
                this.addResult = 'Source "' + data.name + '" added (id: ' + data.id + ')';
                this.addName = ''; this.addUrl = ''; this.addType = ''; this.addMethod = ''; this.addPath = '';
                await this.listSources();
            } catch (e) {
                this.addResult = 'Error: ' + e.message;
            }
        },
        async setEnabled(source, enabled) {
            try {
                const res = await fetch('/api/v1/oracle/sources/' + encodeURIComponent(source.id), {
                    method: 'PATCH',
                    headers: auroraHeaders({ 'Content-Type': 'application/json' }),
                    body: JSON.stringify({ enabled: enabled })
                });
                if (!res.ok) { alert('Failed to ' + (enabled ? 'enable' : 'disable') + ' source'); }
                await this.listSources();
            } catch (e) { alert('Error: ' + e.message); }
        },
        async deleteSource(source) {
            if (!confirm('Delete source "' + (source.name || source.id) + '"?')) return;
            try {
                const res = await fetch('/api/v1/oracle/sources/' + encodeURIComponent(source.id), {
                    method: 'DELETE',
                    headers: auroraHeaders()
                });
                if (!res.ok) { alert('Failed to delete source'); }
                await this.listSources();
            } catch (e) { alert('Error: ' + e.message); }
        },
        async listSources() {
            this.loading = true;
            try {
                const res = await fetch('/api/v1/oracle/sources', { headers: auroraHeaders() });
                const data = await res.json();
                this.sources = (data && data.sources) || [];
            } catch (e) { this.sources = []; }
            this.loading = false;
        },
        async fetch() {
            try {
                const res = await fetch('/api/v1/oracle/fetch', {
                    method: 'POST',
                    headers: auroraHeaders({ 'Content-Type': 'application/json' }),
                    body: JSON.stringify({ source: this.fetchSource })
                });
                this.fetchResult = JSON.stringify(await res.json(), null, 2);
            } catch (e) {
                this.fetchResult = 'Error: ' + e.message;
            }
        },
        async query() {
            try {
                const res = await fetch('/api/v1/oracle/query?source=' + encodeURIComponent(this.querySource) + '&limit=' + this.queryLimit, { headers: auroraHeaders() });
                const data = await res.json();
                this.queryRows = (data && data.data) || [];
            } catch (e) { this.queryRows = []; }
        }
    };
}
