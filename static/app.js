// Setback Game Client

class SetbackGame {
    constructor() {
        this.ws = null;
        this.state = null;
        this.yourSeat = null;
        this.yourHand = [];
        this.yourToken = null;
        this.lastScoreResult = null;
        this.kitty = [];
        this.selectedTrump = null;
        this.selectedKittyCards = new Set();
        this.selectedDiscards = new Set();      // For kitty phase (bid winner)
        this.selectedDrawDiscards = new Set();  // For discard phase (all players)
        this.editingName = false;               // Whether name input is showing

        this.init();
    }

    init() {
        this.connectWebSocket();
        this.setupEventListeners();
        this.loadToken();
    }

    // WebSocket connection
    connectWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws`;

        this.ws = new WebSocket(wsUrl);

        this.ws.onopen = () => {
            console.log('Connected to server');
            if (this.yourToken) {
                this.send({ type: 'rejoin', token: this.yourToken });
            }
        };

        this.ws.onmessage = (event) => {
            const msg = JSON.parse(event.data);
            this.handleMessage(msg);
        };

        this.ws.onclose = () => {
            console.log('Disconnected from server');
            this.showMessage('Disconnected. Reconnecting...', 'error');
            setTimeout(() => this.connectWebSocket(), 2000);
        };

        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
        };
    }

    send(msg) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify(msg));
        }
    }

    loadToken() {
        this.yourToken = localStorage.getItem('setback_token');
    }

    saveToken(token) {
        this.yourToken = token;
        localStorage.setItem('setback_token', token);
    }

    // Message handling
    handleMessage(msg) {
        console.log('Received:', msg);

        switch (msg.type) {
            case 'stateUpdate':
                this.handleStateUpdate(msg);
                break;
            case 'error':
                // Silently handle rejoin failures (stale token) - just clear the token
                if (msg.error.message === 'rejoin_failed') {
                    localStorage.removeItem('setback_token');
                    this.yourToken = null;
                } else {
                    this.showMessage(msg.error.message, 'error');
                }
                break;
            case 'scoreUpdate':
                this.lastScoreResult = msg.scoreResult;
                this.handleScoreUpdate(msg.scoreResult);
                break;
            case 'gameOver':
                this.handleGameOver(msg.winningTeam);
                break;
        }
    }

    handleStateUpdate(msg) {
        const previousState = this.state;
        const previousHouse = this.state?.house;
        this.state = msg.state;

        if (msg.yourSeat !== undefined) {
            this.yourSeat = msg.yourSeat;
        }

        this.yourHand = msg.yourHand || [];
        this.kitty = msg.kitty || [];

        // Reset selections when phase changes
        if (this.state.phase !== 'kitty') {
            this.selectedTrump = null;
            this.selectedKittyCards = new Set();
            this.selectedDiscards = new Set();
        }
        if (this.state.phase !== 'discard') {
            this.selectedDrawDiscards = new Set();
        }

        // Notify player if they became the house
        if (this.yourSeat !== null &&
            this.state.house === this.yourSeat &&
            previousHouse !== this.yourSeat &&
            previousHouse !== undefined) {
            this.showMessage('You are now the house!');
        }

        // Check if a trick was just completed (4 cards in current trick with winner set)
        if (this.state.currentTrick?.cards?.length === 4 &&
            this.state.currentTrick.winner !== undefined &&
            this.state.currentTrick.winner !== null) {
            // Show trick winner notification
            const winnerName = this.getPlayerLabel(this.state.currentTrick.winner);
            const winnerTeam = this.getTeamForSeat(this.state.currentTrick.winner);
            this.showMessage(`${winnerName} (Team ${winnerTeam + 1}) wins the trick!`);
        }

        if (msg.yourToken) {
            this.saveToken(msg.yourToken);
        }

        this.render();
    }

    handleScoreUpdate(result) {
        const resultDiv = document.getElementById('score-result');
        const bidderTeam = result.bidderTeam + 1;
        const madeClass = result.bidMade ? 'made' : 'set';
        const madeText = result.bidMade ? 'Made it!' : 'Set back!';

        let html = `<div class="result-header ${madeClass}">Team ${bidderTeam} bid ${result.bidAmount} - ${madeText}</div>`;

        html += '<div class="points-breakdown">';

        // High
        if (result.highTeam >= 0) {
            html += `<div class="point-row"><span>High</span><span>Team ${result.highTeam + 1}</span></div>`;
        }

        // Low
        if (result.lowTeam >= 0) {
            html += `<div class="point-row"><span>Low</span><span>Team ${result.lowTeam + 1}</span></div>`;
        }

        // Jack
        if (result.jackTeam >= 0) {
            html += `<div class="point-row"><span>Jack</span><span>Team ${result.jackTeam + 1}</span></div>`;
        } else {
            html += `<div class="point-row"><span>Jack</span><span>Not played</span></div>`;
        }

        // Off Jack
        if (result.offJackTeam >= 0) {
            html += `<div class="point-row"><span>Off Jack</span><span>Team ${result.offJackTeam + 1}</span></div>`;
        } else {
            html += `<div class="point-row"><span>Off Jack</span><span>Not played</span></div>`;
        }

        // Game
        if (result.gameTeam >= 0) {
            html += `<div class="point-row"><span>Game (${result.gamePoints[0]}-${result.gamePoints[1]})</span><span>Team ${result.gameTeam + 1}</span></div>`;
        } else {
            html += `<div class="point-row"><span>Game (${result.gamePoints[0]}-${result.gamePoints[1]})</span><span>Tie</span></div>`;
        }

        html += '</div>';

        // Totals
        html += '<div class="totals">';
        const t0Sign = result.team0Change >= 0 ? '+' : '';
        const t1Sign = result.team1Change >= 0 ? '+' : '';
        html += `<div class="point-row"><span>Team 1</span><span>${t0Sign}${result.team0Change}</span></div>`;
        html += `<div class="point-row"><span>Team 2</span><span>${t1Sign}${result.team1Change}</span></div>`;
        html += '</div>';

        resultDiv.innerHTML = html;
    }

    handleGameOver(winningTeam) {
        const winnerDiv = document.getElementById('winner-display');

        // Get team member names
        const teamIndices = winningTeam === 0 ? [0, 2] : [1, 3];
        const player1 = this.getPlayerLabel(teamIndices[0]);
        const player2 = this.getPlayerLabel(teamIndices[1]);

        winnerDiv.innerHTML = `<div>Team ${winningTeam + 1} Wins!</div><div style="font-size: 1.2rem; margin-top: 10px;">${player1} & ${player2}</div>`;

        // Show notification message as well
        this.showMessage(`Game over! ${player1} & ${player2} (Team ${winningTeam + 1}) win!`);
    }

    // Rendering
    render() {
        if (!this.state) return;

        this.renderPlayerIdentity();
        this.renderPhase();
        this.renderScores();
        this.renderPlayers();
        this.renderTrump();
        this.renderKitty();
        this.renderTrick();
        this.renderHand();
        this.renderControls();
    }

    renderPlayerIdentity() {
        const identityBanner = document.getElementById('player-identity');
        const yourIdentity = document.getElementById('your-identity');
        const partnerIdentity = document.getElementById('partner-identity');

        if (this.yourSeat === null || this.yourSeat === undefined) {
            identityBanner.classList.add('hidden');
            return;
        }

        identityBanner.classList.remove('hidden');

        // Get your info
        const yourPlayer = this.state.players[this.yourSeat];
        const yourName = yourPlayer?.name || `Player ${this.yourSeat + 1}`;
        const yourTeam = this.getTeamForSeat(this.yourSeat);
        const teamClass = yourTeam === 0 ? 'team-0-text' : 'team-1-text';

        // Get partner info
        const partnerSeat = this.getPartnerSeat(this.yourSeat);
        const partnerPlayer = this.state.players[partnerSeat];
        const partnerName = partnerPlayer?.name || `Player ${partnerSeat + 1}`;

        yourIdentity.innerHTML = `You: <strong>${yourName}</strong> <span class="seat-num">Seat ${this.yourSeat + 1}</span> <span class="${teamClass}">(Team ${yourTeam + 1})</span>`;
        partnerIdentity.innerHTML = `Partner: <strong>${partnerName}</strong> <span class="seat-num">Seat ${partnerSeat + 1}</span>`;
    }

    getTeamForSeat(seat) {
        return (seat === 0 || seat === 2) ? 0 : 1;
    }

    getPartnerSeat(seat) {
        // Partners are seats 0&2 and 1&3
        const partners = { 0: 2, 2: 0, 1: 3, 3: 1 };
        return partners[seat];
    }

    renderPhase() {
        const phaseDisplay = document.getElementById('phase-display');
        const phaseNames = {
            'lobby': 'Lobby - Waiting for Players',
            'bidding': 'Bidding',
            'kitty': 'Trump Selection & Kitty',
            'discard': 'Discard & Draw',
            'playing': 'Playing',
            'scoring': 'Hand Complete',
            'finished': 'Game Over'
        };
        phaseDisplay.textContent = phaseNames[this.state.phase] || this.state.phase;

        // Trick count
        const trickCount = document.getElementById('trick-count');
        if (this.state.phase === 'playing') {
            trickCount.textContent = `Trick ${this.state.tricksPlayed + 1} of 6`;
            trickCount.style.display = '';
        } else {
            trickCount.style.display = 'none';
        }

        // Target score
        document.getElementById('target-value').textContent = this.state.targetScore || 52;
    }

    renderScores() {
        document.getElementById('team0-score').textContent = this.state.teams[0].score;
        document.getElementById('team1-score').textContent = this.state.teams[1].score;

        // Display games won
        const team0Games = this.state.teams[0].gamesWon || 0;
        const team1Games = this.state.teams[1].gamesWon || 0;
        document.getElementById('team0-games').textContent = `Games: ${team0Games}`;
        document.getElementById('team1-games').textContent = `Games: ${team1Games}`;

        // Display current bid on the team that is bidding (after bidding phase)
        const team0Bid = document.getElementById('team0-bid');
        const team1Bid = document.getElementById('team1-bid');

        // Show bid if bidding is complete and we have a winning bid
        const showBid = this.state.winningBid > 0 &&
            (this.state.phase === 'kitty' || this.state.phase === 'discard' || this.state.phase === 'playing');

        if (showBid) {
            // Determine which team has the bid
            const bidderTeam = this.getTeamForSeat(this.state.bidWinner);
            const bidderName = this.getPlayerLabel(this.state.bidWinner);

            if (bidderTeam === 0) {
                team0Bid.textContent = `Bid: ${this.state.winningBid} (${bidderName})`;
                team0Bid.classList.remove('hidden');
                team1Bid.classList.add('hidden');
            } else {
                team1Bid.textContent = `Bid: ${this.state.winningBid} (${bidderName})`;
                team1Bid.classList.remove('hidden');
                team0Bid.classList.add('hidden');
            }
        } else {
            team0Bid.classList.add('hidden');
            team1Bid.classList.add('hidden');
        }
    }

    renderPlayers() {
        this.state.players.forEach((player, idx) => {
            const seat = document.querySelector(`.seat[data-seat="${idx}"]`);
            const nameEl = seat.querySelector('.player-name');
            const statusEl = seat.querySelector('.player-status');
            const cardBacksEl = seat.querySelector('.card-backs');

            // Show name with house badge, kick/transfer buttons for house
            const isHouse = this.yourSeat === this.state.house;
            const isThisPlayerHouse = idx === this.state.house;
            const canKick = isHouse && idx !== this.yourSeat && player.name; // House can kick other seated players
            const canTransfer = isHouse && idx !== this.yourSeat && player.name && player.connected; // House can transfer to other connected players

            if (player.name) {
                let nameHtml = player.name;
                if (isThisPlayerHouse) {
                    nameHtml += ' <span class="house-badge">House</span>';
                }
                if (canKick) {
                    nameHtml += ` <button class="kick-btn" data-seat="${idx}">Kick</button>`;
                }
                if (canTransfer) {
                    nameHtml += ` <button class="transfer-btn" data-seat="${idx}" title="Make House">🏠</button>`;
                }
                nameEl.innerHTML = nameHtml;
            } else if (!player.connected && this.state.phase !== 'lobby') {
                nameEl.innerHTML = '<em>(open seat)</em>';
            } else {
                nameEl.textContent = '';
            }

            // Add click handler for kick button
            const kickBtn = nameEl.querySelector('.kick-btn');
            if (kickBtn) {
                kickBtn.onclick = (e) => {
                    e.stopPropagation();
                    const seatToKick = parseInt(kickBtn.dataset.seat);
                    const playerName = this.state.players[seatToKick]?.name || `Seat ${seatToKick + 1}`;
                    if (confirm(`Kick ${playerName} from the game?`)) {
                        this.send({ type: 'kickPlayer', seatIndex: seatToKick });
                    }
                };
            }

            // Add click handler for transfer button
            const transferBtn = nameEl.querySelector('.transfer-btn');
            if (transferBtn) {
                transferBtn.onclick = (e) => {
                    e.stopPropagation();
                    const seatToTransfer = parseInt(transferBtn.dataset.seat);
                    const playerName = this.state.players[seatToTransfer]?.name || `Seat ${seatToTransfer + 1}`;
                    if (confirm(`Transfer house ownership to ${playerName}?`)) {
                        this.send({ type: 'transferHouse', seatIndex: seatToTransfer });
                    }
                };
            }

            // Status text (no card counts shown to other players)
            let status = '';
            if (this.state.phase === 'bidding') {
                const bid = this.state.bids.find(b => b.playerIndex === idx);
                if (bid) {
                    status = bid.amount === 0 ? 'Passed' : `Bid ${bid.amount}`;
                }
            } else if (this.state.phase === 'kitty') {
                if (idx === this.state.bidWinner) {
                    status = 'Picking from kitty';
                }
            } else if (this.state.phase === 'discard') {
                if (player.discardComplete) {
                    status = 'Done';
                } else if (player.discardReady) {
                    status = 'Ready';
                } else if (idx === this.state.currentPlayer) {
                    status = 'Discarding...';
                } else {
                    status = 'Selecting...';
                }
            }
            statusEl.textContent = status;

            // Card backs for other players (not your seat)
            if (cardBacksEl && idx !== this.yourSeat) {
                let backs = '';
                for (let i = 0; i < player.cardCount; i++) {
                    backs += '<img class="card-back" src="cards/back.svg" alt="card">';
                }
                cardBacksEl.innerHTML = backs;
            }

            // Highlight current turn
            const isActiveTurn = idx === this.state.currentPlayer &&
                (this.state.phase === 'bidding' || this.state.phase === 'playing' || this.state.phase === 'discard');
            const isKittyTurn = idx === this.state.bidWinner && this.state.phase === 'kitty';
            seat.classList.toggle('current-turn', isActiveTurn || isKittyTurn);
            seat.classList.toggle('dealer', idx === this.state.dealer);
        });

        // Update seat buttons
        document.querySelectorAll('.seat-btn').forEach(btn => {
            const seatIdx = parseInt(btn.dataset.seat);
            const player = this.state.players[seatIdx];
            const taken = player && player.name && player.connected;
            const isYours = seatIdx === this.yourSeat;
            const isOpen = player && !player.name && !player.connected;
            const seatLabel = player?.name || `Seat ${seatIdx + 1}`;

            btn.classList.toggle('taken', taken && !isYours);
            btn.classList.toggle('yours', isYours);

            // In lobby, can switch seats. Mid-game, can only take open seats
            if (this.state.phase === 'lobby') {
                btn.disabled = taken && !isYours;
                btn.textContent = isYours ? `Leave ${seatLabel}` : (taken ? seatLabel : `Seat ${seatIdx + 1}`);
            } else {
                // Mid-game: can only join open seats
                btn.disabled = !isOpen || isYours;
                if (isYours) {
                    btn.textContent = `Your Seat (${seatLabel})`;
                } else if (isOpen) {
                    btn.textContent = `Join Seat ${seatIdx + 1}`;
                } else {
                    btn.textContent = seatLabel;
                }
            }
        });

        // Start button - only house can start the game
        const startBtn = document.getElementById('start-game-btn');
        const allSeated = this.state.players.every(p => p && p.name);
        const isHouse = this.yourSeat === this.state.house;

        // Only show start button to house
        if (isHouse) {
            startBtn.classList.remove('hidden');
            startBtn.disabled = !allSeated || this.state.phase !== 'lobby';

            if (allSeated) {
                startBtn.textContent = 'Start Game';
            } else {
                // List missing players - use player name if set, otherwise seat number
                const missingPlayers = [];
                this.state.players.forEach((p, idx) => {
                    if (!p || !p.name) {
                        missingPlayers.push(`Seat ${idx + 1}`);
                    }
                });
                startBtn.textContent = `Waiting for: ${missingPlayers.join(', ')}`;
            }
        } else {
            startBtn.classList.add('hidden');
        }
    }

    renderTrump() {
        const trumpDisplay = document.getElementById('trump-display');
        if (this.state.trump) {
            const symbols = { spades: '♠', hearts: '♥', diamonds: '♦', clubs: '♣' };
            trumpDisplay.textContent = `Trump: ${symbols[this.state.trump] || this.state.trump}`;
            trumpDisplay.className = this.state.trump;
            trumpDisplay.style.display = '';
        } else {
            trumpDisplay.style.display = 'none';
        }
    }

    renderKitty() {
        const kittyArea = document.getElementById('kitty-area');
        if (!kittyArea) return;

        // Only show kitty area during kitty phase for the bid winner
        if (this.state.phase !== 'kitty' || this.yourSeat !== this.state.bidWinner) {
            kittyArea.classList.add('hidden');
            return;
        }

        kittyArea.classList.remove('hidden');
        const kittyCards = document.getElementById('kitty-cards');
        kittyCards.innerHTML = '';

        // Get trump for highlighting (use selected or confirmed trump)
        const trump = this.state.trump || this.selectedTrump;

        // Render kitty cards
        this.kitty.forEach(card => {
            const cardEl = document.createElement('img');
            cardEl.className = 'card kitty-card';
            cardEl.src = this.getCardImage(card);
            cardEl.alt = card.id;
            cardEl.dataset.cardId = card.id;

            if (this.selectedKittyCards.has(card.id)) {
                cardEl.classList.add('selected');
            }

            // Highlight trump cards after trump is selected
            if (trump && this.isCardTrump(card, trump)) {
                cardEl.classList.add('trump-card');
            }

            cardEl.onclick = () => this.toggleKittyCard(card.id);
            kittyCards.appendChild(cardEl);
        });

        // Update kitty info
        const kittyInfo = document.getElementById('kitty-info');
        const selectedCount = this.selectedKittyCards.size;
        kittyInfo.textContent = `Select cards to take (${selectedCount} selected)`;
    }

    toggleKittyCard(cardId) {
        if (this.selectedKittyCards.has(cardId)) {
            this.selectedKittyCards.delete(cardId);
        } else {
            this.selectedKittyCards.add(cardId);
        }
        this.renderKitty();
        this.renderControls();
    }

    toggleDiscardCard(cardId) {
        if (this.selectedDiscards.has(cardId)) {
            this.selectedDiscards.delete(cardId);
        } else {
            this.selectedDiscards.add(cardId);
        }
        this.renderHand();
        this.renderControls();
    }

    toggleDrawDiscardCard(cardId) {
        if (this.selectedDrawDiscards.has(cardId)) {
            this.selectedDrawDiscards.delete(cardId);
        } else {
            this.selectedDrawDiscards.add(cardId);
        }
        this.renderHand();
        this.renderControls();
    }

    renderTrick() {
        // Clear all trick cards
        document.querySelectorAll('.trick-card').forEach(el => {
            el.innerHTML = '';
            el.classList.remove('winning');
        });

        const winnerDisplay = document.getElementById('trick-winner-display');
        winnerDisplay.textContent = '';

        // Show current trick
        const trick = this.state.currentTrick;
        if (trick && trick.cards.length > 0) {
            trick.cards.forEach(tc => {
                const slot = document.querySelector(`.trick-card[data-seat="${tc.playerIndex}"]`);
                if (slot) {
                    slot.innerHTML = `<img src="${this.getCardImage(tc.card)}" alt="${tc.card.id}">`;
                }
            });

            // If trick is complete, highlight winner
            if (trick.cards.length === 4 && trick.winner !== undefined) {
                const winningSlot = document.querySelector(`.trick-card[data-seat="${trick.winner}"]`);
                if (winningSlot) {
                    winningSlot.classList.add('winning');
                }
                const winnerName = this.state.players[trick.winner]?.name || `Player ${trick.winner + 1}`;
                winnerDisplay.textContent = `${winnerName} wins!`;
            }
        }

        // If no current trick but there's a last trick, show it faded with winner message
        if ((!trick || trick.cards.length === 0) && this.state.lastTrick && this.state.lastTrick.cards.length === 4) {
            // Show last trick faded
            this.state.lastTrick.cards.forEach(tc => {
                const slot = document.querySelector(`.trick-card[data-seat="${tc.playerIndex}"]`);
                if (slot) {
                    slot.innerHTML = `<img src="${this.getCardImage(tc.card)}" alt="${tc.card.id}" style="opacity: 0.5">`;
                }
            });
            // Show winner message from last trick
            if (this.state.lastTrick.winner !== undefined) {
                const winnerName = this.state.players[this.state.lastTrick.winner]?.name || `Player ${this.state.lastTrick.winner + 1}`;
                winnerDisplay.textContent = `${winnerName} wins!`;
            }
        }
    }

    renderHand() {
        const handEl = document.getElementById('hand');
        const handLabel = document.getElementById('hand-label');
        handEl.innerHTML = '';

        if (!this.yourHand || this.yourHand.length === 0) {
            if (this.yourSeat !== null && this.state.phase !== 'lobby') {
                handLabel.textContent = 'Your hand is empty';
            } else if (this.yourSeat === null) {
                handLabel.textContent = 'Join a seat to play';
            } else {
                handLabel.textContent = 'Your Hand';
            }
            return;
        }

        // Update hand label based on phase
        if (this.state.phase === 'kitty' && this.yourSeat === this.state.bidWinner) {
            const minToDiscard = this.yourHand.length - 6;
            if (minToDiscard > 0) {
                const remaining = minToDiscard - this.selectedDiscards.size;
                if (remaining > 0) {
                    handLabel.textContent = `Your Hand (select at least ${remaining} more to discard)`;
                } else {
                    const willDraw = this.selectedDiscards.size - minToDiscard;
                    if (willDraw > 0) {
                        handLabel.textContent = `Your Hand (${this.selectedDiscards.size} to discard, will draw ${willDraw} new)`;
                    } else {
                        handLabel.textContent = `Your Hand (${this.selectedDiscards.size} selected for discard)`;
                    }
                }
            } else {
                // Hand is 6 or less - can still choose to discard and draw
                if (this.selectedDiscards.size > 0) {
                    handLabel.textContent = `Your Hand (${this.selectedDiscards.size} to discard, will draw ${this.selectedDiscards.size} new)`;
                } else {
                    handLabel.textContent = 'Your Hand (select cards to discard, or continue)';
                }
            }
        } else if (this.state.phase === 'discard') {
            const myPlayer = this.yourSeat !== null ? this.state.players[this.yourSeat] : null;
            if (myPlayer && !myPlayer.discardComplete && !myPlayer.discardReady) {
                if (this.selectedDrawDiscards.size > 0) {
                    handLabel.textContent = `Your Hand (${this.selectedDrawDiscards.size} to discard, will draw ${this.selectedDrawDiscards.size} new)`;
                } else {
                    handLabel.textContent = 'Your Hand (select cards to discard, or keep all)';
                }
            } else if (myPlayer?.discardReady) {
                handLabel.textContent = 'Your Hand (discard submitted, waiting...)';
            } else {
                handLabel.textContent = 'Your Hand';
            }
        } else {
            handLabel.textContent = 'Your Hand';
        }

        // Sort hand by suit then rank
        const sorted = [...this.yourHand].sort((a, b) => {
            if (a.suit !== b.suit) return a.suit - b.suit;
            return b.rank - a.rank;
        });

        const isMyTurn = this.yourSeat === this.state.currentPlayer;
        const isPlayPhase = this.state.phase === 'playing';
        const isKittyPhase = this.state.phase === 'kitty' && this.yourSeat === this.state.bidWinner;
        // Allow card selection during discard phase if player hasn't submitted yet
        const myPlayer = this.yourSeat !== null ? this.state.players[this.yourSeat] : null;
        const canSelectDiscards = this.state.phase === 'discard' &&
            myPlayer && !myPlayer.discardComplete && !myPlayer.discardReady;

        // Get trump for highlighting during discard selection (use selected or confirmed trump)
        const trump = this.state.trump || this.selectedTrump;

        sorted.forEach(card => {
            const cardEl = document.createElement('img');
            cardEl.className = 'card';
            cardEl.src = this.getCardImage(card);
            cardEl.alt = card.id;
            cardEl.dataset.cardId = card.id;

            if (isKittyPhase) {
                // During kitty phase, cards can be selected for discard
                cardEl.classList.add('selectable');
                if (this.selectedDiscards.has(card.id)) {
                    cardEl.classList.add('selected-discard');
                }
                // Highlight trump cards to help user identify them
                if (trump && this.isCardTrump(card, trump)) {
                    cardEl.classList.add('trump-card');
                }
                cardEl.onclick = () => this.toggleDiscardCard(card.id);
            } else if (canSelectDiscards) {
                // During discard phase, cards can be selected for discard/draw
                cardEl.classList.add('selectable');
                if (this.selectedDrawDiscards.has(card.id)) {
                    cardEl.classList.add('selected-discard');
                }
                // Highlight trump cards to help user identify them
                if (trump && this.isCardTrump(card, trump)) {
                    cardEl.classList.add('trump-card');
                }
                cardEl.onclick = () => this.toggleDrawDiscardCard(card.id);
            } else {
                const playable = isMyTurn && isPlayPhase && this.canPlayCard(card);
                cardEl.classList.toggle('playable', playable);

                if (playable) {
                    cardEl.onclick = () => this.playCard(card.id);
                }
            }

            handEl.appendChild(cardEl);
        });
    }

    canPlayCard(card) {
        if (!this.state.trump) return true;

        const trump = this.state.trump;
        const cardIsTrump = this.isCardTrump(card, trump);

        // Leading a trick - any card can lead
        if (!this.state.currentTrick || this.state.currentTrick.cards.length === 0) {
            return true;
        }

        const leadSuit = this.state.currentTrick.leadSuit;
        const trumpLed = leadSuit === trump;

        // Check if player has the lead suit
        let hasLeadSuit = false;
        if (trumpLed) {
            // Trump led: check if player has any trump (including Off Jack)
            hasLeadSuit = this.yourHand.some(c => this.isCardTrump(c, trump));
        } else {
            // Non-trump led: check if player has lead suit (Off Jack doesn't count as its native suit)
            hasLeadSuit = this.yourHand.some(c => {
                const suit = this.getSuitName(c.suit);
                return suit === leadSuit && !this.isCardTrump(c, trump);
            });
        }

        if (trumpLed) {
            // Trump led: must play trump if you have it
            if (cardIsTrump) return true;
            return !hasLeadSuit; // Can only play non-trump if no trump in hand
        } else {
            // Non-trump led: must follow suit if you have it
            // Can only play trump or other suits if you don't have the lead suit
            if (hasLeadSuit) {
                // Must follow suit - only cards of lead suit are playable
                const cardSuit = this.getSuitName(card.suit);
                return cardSuit === leadSuit && !cardIsTrump;
            }
            // No lead suit - can play anything (including trump)
            return true;
        }
    }

    // Check if a card is trump (including Off Jack)
    isCardTrump(card, trump) {
        const cardSuit = this.getSuitName(card.suit);
        if (cardSuit === trump) return true;
        // Off Jack: Jack of the same color suit
        if (card.rank === 11) {  // Jack
            const offSuits = { spades: 'clubs', clubs: 'spades', hearts: 'diamonds', diamonds: 'hearts' };
            if (cardSuit === offSuits[trump]) return true;
        }
        return false;
    }

    getSuitName(suit) {
        const suits = ['spades', 'hearts', 'diamonds', 'clubs'];
        return suits[suit];
    }

    // Get a display label for a player - name if set, otherwise "Seat X"
    getPlayerLabel(seatIndex) {
        if (seatIndex < 0 || seatIndex > 3) return 'Unknown';
        const player = this.state.players[seatIndex];
        return player?.name || `Seat ${seatIndex + 1}`;
    }

    getCardImage(card) {
        return `cards/${card.id}.svg`;
    }

    renderControls() {
        const isMyTurn = this.yourSeat === this.state.currentPlayer;
        const isSeated = this.yourSeat !== null;
        const isBidWinner = this.yourSeat === this.state.bidWinner;
        const isHouse = this.yourSeat === this.state.house;

        // Hide all control groups first
        document.getElementById('seat-selection').classList.add('hidden');
        document.getElementById('bid-controls').classList.add('hidden');
        document.getElementById('kitty-controls').classList.add('hidden');
        document.getElementById('discard-controls').classList.add('hidden');
        document.getElementById('waiting-controls').classList.add('hidden');
        document.getElementById('score-controls').classList.add('hidden');
        document.getElementById('gameover-controls').classList.add('hidden');

        // Session controls (name, leave seat, reset) - always visible at bottom
        const sessionControls = document.getElementById('session-controls');
        const leaveSeatBtn = document.getElementById('leave-seat-btn');
        const resetGameBtn = document.getElementById('reset-game-btn');
        const seatSelection = document.getElementById('seat-selection');
        const nameInputArea = document.getElementById('name-input-area');
        const nameDisplay = document.getElementById('name-display');
        const changeNameLink = document.getElementById('change-name-link');
        const currentNameLabel = document.getElementById('current-name-label');
        const saveNameBtn = document.getElementById('save-name-btn');
        const cancelNameBtn = document.getElementById('cancel-name-btn');

        if (isSeated) {
            // Show current name with change link
            const yourPlayer = this.state.players[this.yourSeat];
            const yourName = yourPlayer?.name || 'Unknown';
            currentNameLabel.textContent = `Playing as: ${yourName}`;
            changeNameLink.classList.remove('hidden');

            // Hide input area by default when seated (unless editing)
            if (!this.editingName) {
                nameInputArea.classList.add('hidden');
                nameDisplay.classList.remove('hidden');
            }

            saveNameBtn.classList.remove('hidden');
            cancelNameBtn.classList.remove('hidden');

            // Only show Leave Seat after game has started (not during lobby)
            if (this.state.phase === 'lobby') {
                leaveSeatBtn.classList.add('hidden');
                seatSelection.classList.remove('hidden');
            } else {
                leaveSeatBtn.classList.remove('hidden');
                seatSelection.classList.add('hidden');
            }
        } else {
            // Not seated - show name input and seat selection
            nameDisplay.classList.add('hidden');
            nameInputArea.classList.remove('hidden');
            saveNameBtn.classList.add('hidden');
            cancelNameBtn.classList.add('hidden');
            leaveSeatBtn.classList.add('hidden');
            seatSelection.classList.remove('hidden');
        }

        // Reset button only visible to house
        if (isHouse) {
            resetGameBtn.classList.remove('hidden');
        } else {
            resetGameBtn.classList.add('hidden');
        }

        // Always show session controls
        sessionControls.classList.remove('hidden');

        // Start game button visibility is handled in renderPlayers (only house sees it)

        if (this.state.phase === 'bidding') {
            if (isSeated && isMyTurn) {
                document.getElementById('bid-controls').classList.remove('hidden');
                this.updateBidButtons();
            } else {
                document.getElementById('waiting-controls').classList.remove('hidden');
                const currentPlayerName = this.getPlayerLabel(this.state.currentPlayer);
                document.getElementById('waiting-message').textContent = `Waiting for ${currentPlayerName} to bid...`;
            }
        } else if (this.state.phase === 'kitty') {
            if (isSeated && isBidWinner) {
                document.getElementById('kitty-controls').classList.remove('hidden');
                this.updateKittyControls();
            } else {
                document.getElementById('waiting-controls').classList.remove('hidden');
                const bidWinnerName = this.getPlayerLabel(this.state.bidWinner);
                document.getElementById('waiting-message').textContent = `Waiting for ${bidWinnerName} to select trump and pick from kitty...`;
            }
        } else if (this.state.phase === 'discard') {
            // All players can select discards during discard phase
            const myPlayer = this.state.players[this.yourSeat];
            const hasDiscarded = myPlayer?.discardComplete;
            const hasPendingDiscard = myPlayer?.discardReady;

            if (isSeated && !hasDiscarded && !hasPendingDiscard) {
                // Player hasn't submitted discards yet - show controls
                document.getElementById('discard-controls').classList.remove('hidden');
                this.updateDiscardControls();
            } else {
                // Player has already discarded or submitted pending discards
                document.getElementById('waiting-controls').classList.remove('hidden');
                if (hasDiscarded) {
                    // Build list of players still discarding
                    const waitingFor = this.state.players
                        .filter((p, idx) => p && !p.discardComplete)
                        .map((p, idx) => this.getPlayerLabel(p.seatIndex));
                    if (waitingFor.length > 0) {
                        document.getElementById('waiting-message').textContent = `Waiting for: ${waitingFor.join(', ')}`;
                    } else {
                        document.getElementById('waiting-message').textContent = 'All discards complete...';
                    }
                } else if (hasPendingDiscard) {
                    const currentPlayerName = this.getPlayerLabel(this.state.currentPlayer);
                    document.getElementById('waiting-message').textContent = `Your discards are ready. Waiting for ${currentPlayerName}...`;
                } else {
                    const currentPlayerName = this.getPlayerLabel(this.state.currentPlayer);
                    document.getElementById('waiting-message').textContent = `Waiting for ${currentPlayerName} to discard...`;
                }
            }
        } else if (this.state.phase === 'playing') {
            if (isSeated && isMyTurn) {
                document.getElementById('waiting-controls').classList.remove('hidden');
                document.getElementById('waiting-message').textContent = 'Your turn - click a card to play';
            } else {
                document.getElementById('waiting-controls').classList.remove('hidden');
                const currentPlayerName = this.getPlayerLabel(this.state.currentPlayer);
                document.getElementById('waiting-message').textContent = `Waiting for ${currentPlayerName} to play...`;
            }
        } else if (this.state.phase === 'scoring') {
            document.getElementById('score-controls').classList.remove('hidden');
        } else if (this.state.phase === 'finished') {
            document.getElementById('gameover-controls').classList.remove('hidden');
        }
    }

    updateKittyControls() {
        // Update trump selection buttons
        document.querySelectorAll('.trump-btn').forEach(btn => {
            const suit = btn.dataset.suit;
            btn.classList.toggle('selected', this.selectedTrump === suit || this.state.trump === suit);
        });

        // Check if we can take from kitty
        const takeKittyBtn = document.getElementById('take-kitty-btn');
        takeKittyBtn.disabled = this.kitty.length === 0;
        if (this.selectedKittyCards.size > 0) {
            takeKittyBtn.textContent = `Take ${this.selectedKittyCards.size} Card(s)`;
        } else {
            takeKittyBtn.textContent = 'Skip Kitty';
        }

        // Calculate how many cards we need to discard at minimum
        const currentHandSize = this.yourHand.length;
        const minToDiscard = currentHandSize > 6 ? currentHandSize - 6 : 0;

        // Check if we can finalize (discard)
        // Allow discarding MORE than minimum - server will deal cards back to 6
        const finalizeBtn = document.getElementById('finalize-kitty-btn');
        const trumpSelected = this.selectedTrump || this.state.trump;
        const kittyEmpty = this.kitty.length === 0;
        const enoughDiscards = this.selectedDiscards.size >= minToDiscard;
        const notDiscardingAll = this.selectedDiscards.size < currentHandSize; // Must keep at least 1 card
        const canFinalize = trumpSelected && kittyEmpty && enoughDiscards && notDiscardingAll;
        finalizeBtn.disabled = !canFinalize;

        // Show/hide Discard Non-Trump button based on whether trump is selected
        const discardNonTrumpBtn = document.getElementById('discard-non-trump-btn');
        if (trumpSelected && kittyEmpty) {
            discardNonTrumpBtn.classList.remove('hidden');
        } else {
            discardNonTrumpBtn.classList.add('hidden');
        }

        // Update button text based on whether player will draw new cards
        const willDraw = this.selectedDiscards.size > minToDiscard ? this.selectedDiscards.size - minToDiscard : 0;
        if (willDraw > 0) {
            finalizeBtn.textContent = `Discard ${this.selectedDiscards.size} & Draw ${willDraw}`;
        } else {
            finalizeBtn.textContent = 'Continue';
        }

        // Update status message
        const kittyStatus = document.getElementById('kitty-status');
        if (!trumpSelected) {
            kittyStatus.textContent = 'Step 1: Select a trump suit';
        } else if (this.kitty.length > 0) {
            if (this.selectedKittyCards.size > 0) {
                kittyStatus.textContent = `Step 2: Selected ${this.selectedKittyCards.size} card(s). Click "Take" or select more, then "Skip" to finish.`;
            } else {
                kittyStatus.textContent = 'Step 2: Select cards from kitty to take, or click "Skip Kitty"';
            }
        } else if (minToDiscard > 0 && this.selectedDiscards.size < minToDiscard) {
            const remaining = minToDiscard - this.selectedDiscards.size;
            kittyStatus.textContent = `Step 3: Select at least ${remaining} more card(s) to discard`;
        } else if (willDraw > 0) {
            kittyStatus.textContent = `Ready! Will discard ${this.selectedDiscards.size} and draw ${willDraw} new cards`;
        } else {
            kittyStatus.textContent = 'Ready! Click "Continue" to begin';
        }
    }

    updateDiscardControls() {
        const discardStatus = document.getElementById('discard-status');
        const discardInfo = document.getElementById('discard-info');
        const confirmBtn = document.getElementById('confirm-discard-btn');
        const discardAllBtn = document.getElementById('discard-all-btn');
        const selectedCount = this.selectedDrawDiscards.size;
        const isMyTurn = this.yourSeat === this.state.currentPlayer;

        if (selectedCount === 0) {
            confirmBtn.textContent = 'Keep All';
        } else {
            confirmBtn.textContent = `Discard ${selectedCount} & Draw`;
        }

        // Discard All button always has same text
        discardAllBtn.textContent = 'Discard All';

        if (isMyTurn) {
            discardStatus.textContent = "Your turn to discard";
            if (selectedCount === 0) {
                discardInfo.textContent = 'Select cards to discard and draw new ones, or keep your hand.';
            } else {
                discardInfo.textContent = `Discard ${selectedCount} card(s) and draw ${selectedCount} new card(s).`;
            }
        } else {
            discardStatus.textContent = "Select your discards now";
            const currentPlayerName = this.getPlayerLabel(this.state.currentPlayer);
            if (selectedCount === 0) {
                discardInfo.textContent = `Select cards to discard while waiting. ${currentPlayerName} is discarding first.`;
            } else {
                discardInfo.textContent = `${selectedCount} card(s) selected. Submit when ready - your draw happens in turn order.`;
            }
        }
    }

    updateBidButtons() {
        const highBid = Math.max(0, ...this.state.bids.map(b => b.amount));
        const infoEl = document.getElementById('current-high-bid');

        if (highBid > 0) {
            infoEl.textContent = `Current high bid: ${highBid}`;
        } else {
            infoEl.textContent = 'No bids yet - you can bid or pass';
        }

        // Check if this is dealer and everyone passed
        const isDealer = this.yourSeat === this.state.dealer;
        const everyonePassed = this.state.bids.length === 3 && highBid === 0;

        document.querySelectorAll('.bid-btn').forEach(btn => {
            const bidAmount = parseInt(btn.dataset.bid);

            if (isDealer && everyonePassed && bidAmount === 0) {
                // Dealer can't pass if everyone else passed
                btn.disabled = true;
                btn.title = 'You must bid (dealer is stuck)';
            } else if (bidAmount === 0) {
                btn.disabled = false;
            } else {
                btn.disabled = bidAmount <= highBid;
            }
        });
    }

    // Event listeners
    setupEventListeners() {
        // Change name link - shows the input field
        document.getElementById('change-name-link').onclick = (e) => {
            e.preventDefault();
            this.editingName = true;
            document.getElementById('name-display').classList.add('hidden');
            document.getElementById('name-input-area').classList.remove('hidden');
            // Pre-fill with current name
            const yourPlayer = this.state.players[this.yourSeat];
            document.getElementById('player-name').value = yourPlayer?.name || '';
            document.getElementById('player-name').focus();
        };

        // Save name button
        document.getElementById('save-name-btn').onclick = () => {
            const nameInput = document.getElementById('player-name');
            const name = nameInput.value.trim();
            if (name) {
                this.send({ type: 'changeName', playerName: name });
            }
            this.editingName = false;
            this.render();
        };

        // Cancel name edit button
        document.getElementById('cancel-name-btn').onclick = () => {
            this.editingName = false;
            this.render();
        };

        // Leave seat button
        document.getElementById('leave-seat-btn').onclick = () => {
            this.leaveSeat();
        };

        // Reset game button (house only)
        document.getElementById('reset-game-btn').onclick = () => {
            if (confirm('Reset the game? All progress will be lost.')) {
                this.send({ type: 'resetGame' });
            }
        };

        document.querySelectorAll('.seat-btn').forEach(btn => {
            btn.onclick = () => {
                const seatIdx = parseInt(btn.dataset.seat);
                if (seatIdx === this.yourSeat) {
                    this.leaveSeat();
                } else {
                    this.joinSeat(seatIdx);
                }
            };
        });

        document.getElementById('start-game-btn').onclick = () => {
            this.send({ type: 'startGame' });
        };

        document.querySelectorAll('.bid-btn').forEach(btn => {
            btn.onclick = () => {
                const amount = parseInt(btn.dataset.bid);
                this.send({ type: 'placeBid', amount: amount });
            };
        });

        document.getElementById('new-hand-btn').onclick = () => {
            this.send({ type: 'newHand' });
        };

        document.getElementById('new-game-btn').onclick = () => {
            this.send({ type: 'newHand' });
        };

        // Kitty controls
        document.querySelectorAll('.trump-btn').forEach(btn => {
            btn.onclick = () => {
                this.selectedTrump = btn.dataset.suit;
                this.send({ type: 'selectTrump', trumpSuit: this.selectedTrump });
            };
        });

        document.getElementById('take-kitty-btn').onclick = () => {
            const cardIds = Array.from(this.selectedKittyCards);
            this.send({ type: 'takeKitty', cardIds: cardIds });
            // Cards will be moved to hand on next state update
            this.selectedKittyCards.clear();
        };

        // Discard Non-Trump button - selects all non-trump cards for discard
        document.getElementById('discard-non-trump-btn').onclick = () => {
            const trump = this.state.trump || this.selectedTrump;
            if (!trump) return;

            // Select all non-trump cards in hand for discard
            this.selectedDiscards.clear();
            this.yourHand.forEach(card => {
                if (!this.isCardTrump(card, trump)) {
                    this.selectedDiscards.add(card.id);
                }
            });
            this.renderHand();
            this.renderControls();
        };

        document.getElementById('finalize-kitty-btn').onclick = () => {
            const cardIds = Array.from(this.selectedDiscards);
            this.send({ type: 'discard', cardIds: cardIds });
            this.selectedDiscards.clear();
        };

        // Discard phase controls
        document.getElementById('confirm-discard-btn').onclick = () => {
            const cardIds = Array.from(this.selectedDrawDiscards);
            this.send({ type: 'discardDraw', cardIds: cardIds });
            this.selectedDrawDiscards.clear();
        };

        // Discard All button - discards all 6 cards and draws 6 new
        document.getElementById('discard-all-btn').onclick = () => {
            const allCardIds = this.yourHand.map(card => card.id);
            this.send({ type: 'discardDraw', cardIds: allCardIds });
            this.selectedDrawDiscards.clear();
        };

        // Discard Non-Trump button for discard phase - selects all non-trump cards
        document.getElementById('discard-non-trump-draw-btn').onclick = () => {
            const trump = this.state.trump;
            if (!trump) return;

            // Select all non-trump cards in hand for discard
            this.selectedDrawDiscards.clear();
            this.yourHand.forEach(card => {
                if (!this.isCardTrump(card, trump)) {
                    this.selectedDrawDiscards.add(card.id);
                }
            });
            this.renderHand();
            this.renderControls();
        };
    }

    joinSeat(seatIndex) {
        const nameInput = document.getElementById('player-name');
        const defaultName = seatIndex !== null ? `Player ${seatIndex + 1}` : 'Player';
        const name = nameInput.value.trim() || defaultName;

        const msg = {
            type: 'joinTable',
            playerName: name
        };

        // If seatIndex is null, server will auto-assign
        if (seatIndex !== null) {
            msg.seatIndex = seatIndex;
        }

        this.send(msg);
    }

    leaveSeat() {
        this.send({ type: 'leaveSeat' });
        this.yourSeat = null;
        this.yourHand = [];
        this.yourToken = null;
        localStorage.removeItem('setback_token');
    }

    playCard(cardId) {
        this.send({ type: 'playCard', cardId: cardId });
    }

    showMessage(text, type = 'info') {
        const area = document.getElementById('message-area');
        const msg = document.createElement('div');
        msg.className = `message ${type}`;
        msg.textContent = text;
        area.appendChild(msg);

        setTimeout(() => msg.remove(), 4000);
    }
}

// Start the game when page loads
window.onload = () => {
    window.game = new SetbackGame();
};
