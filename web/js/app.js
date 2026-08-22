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
        async init() {
            await this.loadHistory();
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
        stats: { lotteries: 0, votes: 0, candidates: 0, sessions: 0 },
        activity: [],
        loading: true,
        async init() {
            await Promise.all([
                this.loadLotteries(),
                this.loadVoting()
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
        }
    };
}
