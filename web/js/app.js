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

function votingApp() {
    return {
        title: '',
        ownerKey: '',
        result: '',
        proposals: [],
        loading: true,
        async init() {
            await this.loadProposals();
        },
        async loadProposals() {
            this.loading = true;
            try {
                const res = await fetch('/api/v1/voting/candidates', { headers: auroraHeaders() });
                this.proposals = await res.json();
            } catch (e) {
                console.error(e);
            }
            this.loading = false;
        },
        async createVote() {
            try {
                const res = await fetch('/api/v1/voting/create', {
                    method: 'POST',
                    headers: auroraHeaders({ 'Content-Type': 'application/json' }),
                    body: JSON.stringify({
                        title: this.title,
                        owner: this.ownerKey
                    })
                });
                const data = await res.json();
                this.result = JSON.stringify(data, null, 2);
                await this.loadProposals();
                this.title = '';
            } catch (e) {
                this.result = 'Error: ' + e.message;
            }
        },
        async castVote(proposalId, vote) {
            alert(`Vote "${vote}" cast for proposal ${proposalId}`);
        }
    };
}
