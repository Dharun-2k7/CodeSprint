// API Configuration
const API_BASE = '/api';
let authToken = localStorage.getItem('authToken');
let currentUser = null;
let currentContestId = null;
let currentProblemId = null;
let leaderboardInterval = null;

// Monaco Editor
let monacoEditor = null;
let monacoReady = false;
const MONACO_VS_BASE = 'https://cdnjs.cloudflare.com/ajax/libs/monaco-editor/0.50.0/min/vs';

// Initialize app
document.addEventListener('DOMContentLoaded', () => {
    checkAuth();
    loadContests();
    setupEventListeners();
    initMonacoEditor();
});

// Authentication
function checkAuth() {
    if (authToken) {
        // Token exists, show user info
        const userData = localStorage.getItem('userData');
        if (userData) {
            currentUser = JSON.parse(userData);
            updateUIForAuth();
        }
    }
}

function updateUIForAuth() {
    document.getElementById('login-btn').style.display = 'none';
    document.getElementById('signup-btn').style.display = 'none';
    document.getElementById('logout-btn').style.display = 'block';
    // Only show admin button if user is admin
    if (currentUser && currentUser.is_admin) {
        document.getElementById('admin-btn').style.display = 'block';
    } else {
        document.getElementById('admin-btn').style.display = 'none';
    }
    document.getElementById('user-info').style.display = 'block';
    document.getElementById('user-info').textContent = `Welcome, ${currentUser.name}`;
}

function setupEventListeners() {
    document.getElementById('login-btn').addEventListener('click', () => showView('login'));
    document.getElementById('signup-btn').addEventListener('click', () => showView('signup'));
    document.getElementById('logout-btn').addEventListener('click', logout);
    document.getElementById('admin-btn').addEventListener('click', () => {
        showView('admin');
        loadAdminContests();
    });
    document.getElementById('login-form').addEventListener('submit', handleLogin);
    document.getElementById('signup-form').addEventListener('submit', handleSignup);
    document.getElementById('run-btn').addEventListener('click', runCode);
    document.getElementById('submit-btn').addEventListener('click', () => submitCode(currentProblemId));
    document.getElementById('back-to-contest-btn').addEventListener('click', () => showContestProblemsView());

    document.getElementById('language-select')?.addEventListener('change', (e) => {
        setMonacoLanguage(e.target.value);
    });
    
    // Forgot password
    document.getElementById('forgot-password-link')?.addEventListener('click', (e) => {
        e.preventDefault();
        showView('forgot-password');
    });
    document.getElementById('back-to-login-link')?.addEventListener('click', (e) => {
        e.preventDefault();
        showView('login');
    });
    document.getElementById('forgot-password-form')?.addEventListener('submit', handleForgotPassword);
    document.getElementById('reset-password-form')?.addEventListener('submit', handleResetPassword);
    document.getElementById('login-2fa-btn')?.addEventListener('click', handleLogin2FA);
    
    // Admin forms
    document.getElementById('create-contest-form')?.addEventListener('submit', handleCreateContest);
    document.getElementById('edit-contest-form')?.addEventListener('submit', handleUpdateContest);
    document.getElementById('edit-contest-standings-frozen')?.addEventListener('change', (e) => {
        document.getElementById('freeze-time-container').style.display = e.target.checked ? 'block' : 'none';
    });
    
    // Handle browser back/forward
    window.addEventListener('popstate', handlePopState);
}

function getMonacoLanguageId(langValue) {
    // Monaco uses `c`, `cpp`, `python` language IDs.
    if (langValue === 'cpp') return 'cpp';
    if (langValue === 'python3') return 'python';
    return 'c';
}

function setMonacoLanguage(langValue) {
    if (!monacoEditor) return;
    const model = monacoEditor.getModel();
    if (!model) return;
    const langId = getMonacoLanguageId(langValue);
    monaco.editor.setModelLanguage(model, langId);
}

function initMonacoEditor() {
    const container = document.getElementById('monaco-editor');
    const textarea = document.getElementById('code-editor');
    if (!container || !window.require) return;

    if (monacoEditor) return;

    window.require.config({ paths: { vs: MONACO_VS_BASE }});
    window.require(['vs/editor/editor.main'], () => {
        monacoEditor = monaco.editor.create(container, {
            value: textarea?.value || '',
            language: 'c',
            theme: 'vs-dark',
            automaticLayout: true,
            minimap: { enabled: false }
        });
        monacoReady = true;

        // Keep the hidden textarea in sync for any legacy code paths.
        monacoEditor.onDidChangeModelContent(() => {
            if (textarea) textarea.value = monacoEditor.getValue();
        });

        setMonacoLanguage(document.getElementById('language-select')?.value || 'c');
    });
}

function getCodeFromEditor() {
    if (monacoEditor) return monacoEditor.getValue();
    const textarea = document.getElementById('code-editor');
    return textarea ? textarea.value : '';
}

function setCodeToEditor(value) {
    if (monacoEditor) {
        monacoEditor.setValue(value || '');
        return;
    }
    const textarea = document.getElementById('code-editor');
    if (textarea) textarea.value = value || '';
}

function showView(viewName) {
    document.getElementById('home-view').style.display = viewName === 'home' ? 'block' : 'none';
    document.getElementById('login-view').style.display = viewName === 'login' ? 'block' : 'none';
    document.getElementById('signup-view').style.display = viewName === 'signup' ? 'block' : 'none';
    document.getElementById('contest-view').style.display = viewName === 'contest' ? 'block' : 'none';
    document.getElementById('admin-view').style.display = viewName === 'admin' ? 'block' : 'none';
    document.getElementById('forgot-password-view').style.display = viewName === 'forgot-password' ? 'block' : 'none';
    
    // Reset 2FA section
    if (viewName !== 'login') {
        document.getElementById('login-2fa-section').style.display = 'none';
        document.getElementById('reset-password-section').style.display = 'none';
    }
}

async function handleLogin(e) {
    e.preventDefault();
    const email = document.getElementById('login-email').value;
    const password = document.getElementById('login-password').value;

    try {
        const response = await fetch(`${API_BASE}/login`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email, password })
        });

        const data = await response.json();
        if (response.ok) {
            // Check if 2FA is required
            if (data.requires_2fa) {
                // Request OTP
                const otpResponse = await fetch(`${API_BASE}/request-2fa-otp`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ email, password })
                });
                const otpData = await otpResponse.json();
                if (otpResponse.ok) {
                    alert(`OTP sent to your email${otpData.otp ? '. OTP: ' + otpData.otp : ''}`);
                    document.getElementById('login-2fa-section').style.display = 'block';
                } else {
                    alert(otpData.error || 'Failed to send OTP');
                }
                return;
            }
            
            // Normal login
            authToken = data.token;
            currentUser = data.user;
            localStorage.setItem('authToken', authToken);
            localStorage.setItem('userData', JSON.stringify(currentUser));
            updateUIForAuth();
            showView('home');
            loadContests();
        } else {
            alert(data.error || 'Login failed');
        }
    } catch (error) {
        alert('Error: ' + error.message);
    }
}

async function handleLogin2FA() {
    const email = document.getElementById('login-email').value;
    const otp = document.getElementById('login-otp').value;

    if (!otp || otp.length !== 6) {
        alert('Please enter a valid 6-digit OTP');
        return;
    }

    try {
        const response = await fetch(`${API_BASE}/login-2fa`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email, otp })
        });

        const data = await response.json();
        if (response.ok) {
            authToken = data.token;
            currentUser = data.user;
            localStorage.setItem('authToken', authToken);
            localStorage.setItem('userData', JSON.stringify(currentUser));
            updateUIForAuth();
            showView('home');
            loadContests();
            document.getElementById('login-2fa-section').style.display = 'none';
        } else {
            alert(data.error || 'Invalid OTP');
        }
    } catch (error) {
        alert('Error: ' + error.message);
    }
}

async function handleForgotPassword(e) {
    e.preventDefault();
    const email = document.getElementById('forgot-email').value;

    try {
        const response = await fetch(`${API_BASE}/forgot-password`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email })
        });

        const data = await response.json();
        if (response.ok) {
            alert(`OTP sent to your email${data.otp ? '. OTP: ' + data.otp : ''}`);
            document.getElementById('reset-password-section').style.display = 'block';
        } else {
            alert(data.error || 'Failed to send OTP');
        }
    } catch (error) {
        alert('Error: ' + error.message);
    }
}

async function handleResetPassword(e) {
    e.preventDefault();
    const otp = document.getElementById('reset-otp').value;
    const password = document.getElementById('reset-new-password').value;

    try {
        const response = await fetch(`${API_BASE}/reset-password`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ otp, password })
        });

        const data = await response.json();
        if (response.ok) {
            alert('Password reset successfully! Please login.');
            showView('login');
        } else {
            alert(data.error || 'Failed to reset password');
        }
    } catch (error) {
        alert('Error: ' + error.message);
    }
}

async function handleSignup(e) {
    e.preventDefault();
    const name = document.getElementById('signup-name').value;
    const email = document.getElementById('signup-email').value;
    const password = document.getElementById('signup-password').value;

    try {
        const response = await fetch(`${API_BASE}/signup`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name, email, password })
        });

        const data = await response.json();
        if (response.ok) {
            authToken = data.token;
            currentUser = data.user;
            localStorage.setItem('authToken', authToken);
            localStorage.setItem('userData', JSON.stringify(currentUser));
            updateUIForAuth();
            showView('home');
            loadContests();
        } else {
            alert(data.error || 'Signup failed');
        }
    } catch (error) {
        alert('Error: ' + error.message);
    }
}

function logout() {
    authToken = null;
    currentUser = null;
    localStorage.removeItem('authToken');
    localStorage.removeItem('userData');
    document.getElementById('login-btn').style.display = 'block';
    document.getElementById('signup-btn').style.display = 'block';
    document.getElementById('logout-btn').style.display = 'none';
    document.getElementById('admin-btn').style.display = 'none';
    document.getElementById('user-info').style.display = 'none';
    showView('home');
    if (leaderboardInterval) {
        clearInterval(leaderboardInterval);
    }
}

// Contests
async function loadContests() {
    try {
        const response = await fetch(`${API_BASE}/contests`);
        const contests = await response.json();
        displayContests(contests);
    } catch (error) {
        console.error('Error loading contests:', error);
    }
}

function displayContests(contests) {
    const container = document.getElementById('contests-list');
    container.innerHTML = '<h3>Available Contests</h3>';
    
    if (contests.length === 0) {
        container.innerHTML += '<p>No contests available</p>';
        return;
    }

    contests.forEach(contest => {
        const card = document.createElement('div');
        card.className = 'card contest-card';
        card.innerHTML = `
            <div class="card-body">
                <h5 class="card-title">${contest.title}</h5>
                <p class="card-text">
                    Start: ${new Date(contest.start_time).toLocaleString()}<br>
                    End: ${new Date(contest.end_time).toLocaleString()}
                </p>
            </div>
        `;
        card.addEventListener('click', () => openContest(contest.id));
        container.appendChild(card);
    });
}

async function openContest(contestId) {
    if (!authToken) {
        alert('Please login to view contests');
        showView('login');
        return;
    }

    currentContestId = contestId;
    showView('contest');
    
    try {
        const [contest, problems] = await Promise.all([
            fetch(`${API_BASE}/contest/${contestId}`).then(r => r.json()),
            fetch(`${API_BASE}/problems?contest_id=${contestId}`, {
                headers: { 'Authorization': `Bearer ${authToken}` }
            }).then(r => r.json())
        ]);

        document.getElementById('contest-title').textContent = contest.title;
        displayProblems(problems);
        loadLeaderboard(contestId);
        
        // Auto-refresh leaderboard every 30 seconds (optimized for performance)
        if (leaderboardInterval) {
            clearInterval(leaderboardInterval);
        }
        // Check if contest is still active before polling
        const now = new Date();
        const endTime = new Date(contest.end_time);
        if (endTime > now) {
            leaderboardInterval = setInterval(() => {
                const checkNow = new Date();
                if (new Date(contest.end_time) > checkNow) {
                    loadLeaderboard(contestId);
                } else {
                    clearInterval(leaderboardInterval);
                }
            }, 30000); // 30 seconds instead of 10
        }
    } catch (error) {
        console.error('Error loading contest:', error);
    }
}

function displayProblems(problems) {
    const container = document.getElementById('problems-list');
    container.innerHTML = '<h4>Problems</h4>';
    
    problems.forEach(problem => {
        const card = document.createElement('div');
        card.className = 'card problem-card mb-2';
        card.style.cursor = 'pointer';
        card.innerHTML = `
            <div class="card-body">
                <h5 class="card-title">${problem.title}</h5>
                <p class="card-text">Time: ${problem.time_limit}ms | Memory: ${problem.memory_limit}MB</p>
            </div>
        `;
        card.addEventListener('click', () => {
            // Update URL without page reload
            const newUrl = `/contest/${currentContestId}/problem/${problem.id}`;
            window.history.pushState({ contestId: currentContestId, problemId: problem.id }, '', newUrl);
            loadProblemInIDE(problem.id);
        });
        container.appendChild(card);
    });
}

function showContestProblemsView() {
    document.getElementById('contest-problems-view').style.display = 'block';
    document.getElementById('problem-view').style.display = 'none';
    document.getElementById('back-to-contest-btn').style.display = 'none';
    // Update URL
    const newUrl = `/contest/${currentContestId}`;
    window.history.pushState({ contestId: currentContestId }, '', newUrl);
}

function handlePopState(event) {
    if (event.state && event.state.contestId) {
        currentContestId = event.state.contestId;
        if (event.state.problemId) {
            loadProblemInIDE(event.state.problemId);
        } else {
            showContestProblemsView();
        }
    }
}

async function loadProblemInIDE(problemId) {
    currentProblemId = problemId;
    
    // Show problem view, hide contest problems view
    document.getElementById('contest-problems-view').style.display = 'none';
    document.getElementById('problem-view').style.display = 'block';
    document.getElementById('back-to-contest-btn').style.display = 'block';
    
    try {
        const response = await fetch(`${API_BASE}/problem/${problemId}`);
        const problem = await response.json();
        
        document.getElementById('problem-title').textContent = problem.title;
        document.getElementById('problem-statement').innerHTML = problem.statement.replace(/\n/g, '<br>');
        
        // Load sample testcases
        const testcasesResponse = await fetch(`${API_BASE}/testcases?problem_id=${problemId}`);
        const testcases = await testcasesResponse.json();
        const sampleContainer = document.getElementById('sample-testcases');
        
        // Filter only sample testcases
        const sampleTestcases = testcases.filter(tc => tc.is_sample);
        
        if (sampleTestcases.length > 0) {
            let html = '<h5>Sample Test Cases</h5>';
            sampleTestcases.forEach((tc, idx) => {
                html += `
                    <div class="card mb-2">
                        <div class="card-body">
                            <strong>Input ${idx + 1}:</strong>
                            <pre class="bg-light p-2 border rounded">${escapeHtml(tc.input)}</pre>
                            <strong>Expected Output ${idx + 1}:</strong>
                            <pre class="bg-light p-2 border rounded">${escapeHtml(tc.expected_output)}</pre>
                        </div>
                    </div>
                `;
            });
            sampleContainer.innerHTML = html;
        } else {
            sampleContainer.innerHTML = '<p class="text-muted">No sample testcases available</p>';
        }
        
        // Clear previous code and output
        setCodeToEditor('');
        document.getElementById('run-output').style.display = 'none';
        
        // Scroll to top
        document.getElementById('problem-panel').scrollTop = 0;
    } catch (error) {
        console.error('Error loading problem:', error);
        alert('Failed to load problem. Please try again.');
    }
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function showSubmissionForm(problemId) {
    const formDiv = document.getElementById('submission-form');
    formDiv.innerHTML = `
        <h4>Submit Solution</h4>
        <form id="code-submission-form">
            <div class="mb-3">
                <label for="language" class="form-label">Language</label>
                <select class="form-select" id="language" required>
                    <option value="c">C</option>
                    <option value="cpp">C++</option>
                    <option value="python3">Python 3</option>
                </select>
            </div>
            <div class="mb-3">
                <label for="code" class="form-label">Code</label>
                <textarea class="form-control code-editor" id="code" required></textarea>
            </div>
            <button type="submit" class="btn btn-primary">Submit</button>
        </form>
    `;
    formDiv.style.display = 'block';
    
    document.getElementById('code-submission-form').addEventListener('submit', (e) => {
        e.preventDefault();
        submitCode(problemId);
    });
}

async function runCode() {
    if (!currentProblemId) {
        alert('Please select a problem first');
        return;
    }

    const language = document.getElementById('language-select').value;
    const code = getCodeFromEditor();

    if (!code.trim()) {
        alert('Please write some code first');
        return;
    }

    // Language is already in correct format (python3 from select)

    // Loading state
    const runBtn = document.getElementById('run-btn');
    const submitBtn = document.getElementById('submit-btn');
    runBtn.disabled = true;
    submitBtn.disabled = true;
    setMonacoLanguage(language);

    const outputPanel = document.getElementById('run-output');
    const outputContent = document.getElementById('output-content');
    outputPanel.style.display = 'block';
    outputContent.textContent = 'Running sample(s)...';
    outputContent.style.color = '#212529';

    try {
        const response = await fetch(`${API_BASE}/run`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${authToken}`
            },
            body: JSON.stringify({
                problem_id: currentProblemId,
                language: language,
                code: code
            })
        });

        const data = await response.json();
        if (response.ok && data.success) {
            let outputText = `Verdict: ${data.verdict}\nRuntime: ${data.runtime_ms}ms\nSamples: ${data.sample_count}\n\n`;
            if (Array.isArray(data.test_results)) {
                data.test_results.forEach((tr, idx) => {
                    outputText += `Sample ${idx + 1} (TC ${tr.testcase_id})\n`;
                    outputText += `Status: ${tr.status} | Matches: ${tr.matches ? 'YES' : 'NO'} | Runtime: ${tr.runtime_ms}ms\n\n`;
                    outputText += `Output:\n${tr.stdout || '(empty)'}\n\n`;
                    outputText += `Expected:\n${tr.expected || '(empty)'}\n\n`;
                    outputText += '----------------------------------------\n';
                });
            }

            outputContent.textContent = outputText;
            outputContent.style.color = data.verdict === 'AC' ? '#28a745' : '#dc3545';
        } else {
            outputContent.textContent = `Error: ${data.error || 'Failed to run code'}`;
            outputContent.style.color = '#dc3545';
        }
    } catch (error) {
        alert('Error: ' + error.message);
        outputContent.textContent = `Error: ${error.message}`;
        outputContent.style.color = '#dc3545';
    } finally {
        runBtn.disabled = false;
        submitBtn.disabled = false;
    }
}

async function submitCode(problemId) {
    if (!problemId) {
        alert('Please select a problem first');
        return;
    }

    const language = document.getElementById('language-select').value;
    const code = getCodeFromEditor();

    if (!code.trim()) {
        alert('Please write some code first');
        return;
    }

    // Normalize language for backend
    let normalizedLang = language;
    if (language === 'python') {
        normalizedLang = 'python3';
    }

    // Loading state
    const runBtn = document.getElementById('run-btn');
    const submitBtn = document.getElementById('submit-btn');
    runBtn.disabled = true;
    submitBtn.disabled = true;
    setMonacoLanguage(language);

    const outputPanel = document.getElementById('run-output');
    const outputContent = document.getElementById('output-content');
    outputPanel.style.display = 'block';
    outputContent.textContent = 'Submitting and running against all testcases...';
    outputContent.style.color = '#212529';

    try {
        const response = await fetch(`${API_BASE}/submission`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${authToken}`
            },
            body: JSON.stringify({
                problem_id: problemId,
                contest_id: currentContestId,
                language: normalizedLang,
                code: code
            })
        });

        const data = await response.json();
        if (response.ok) {
            if (data.success) {
                let outputText = `Verdict: ${data.verdict}\nScore: ${data.score}\nRuntime: ${data.runtime_ms}ms\nSubmission ID: ${data.submission_id}\n\n`;
                if (Array.isArray(data.test_results)) {
                    data.test_results.forEach((tr, idx) => {
                        outputText += `TC ${tr.testcase_id} - ${tr.status} | Matches: ${tr.matches ? 'YES' : 'NO'} | Runtime: ${tr.runtime_ms}ms\n`;
                        outputText += `Output:\n${tr.stdout || '(empty)'}\n\n`;
                        outputText += `Expected:\n${tr.expected || '(empty)'}\n\n`;
                        outputText += '----------------------------------------\n';
                    });
                }
                outputContent.textContent = outputText;
                outputContent.style.color = data.verdict === 'AC' ? '#28a745' : '#dc3545';
                if (currentContestId) loadLeaderboard(currentContestId);
            } else {
                outputContent.textContent = `Error: ${data.error || 'Submission failed'}`;
                outputContent.style.color = '#dc3545';
            }
        } else {
            outputContent.textContent = `Error: ${data.error || 'Submission failed'}`;
            outputContent.style.color = '#dc3545';
        }
    } catch (error) {
        alert('Error: ' + error.message);
        outputContent.textContent = `Error: ${error.message}`;
        outputContent.style.color = '#dc3545';
    } finally {
        runBtn.disabled = false;
        submitBtn.disabled = false;
    }
}

async function pollSubmissionResult(submissionId) {
    const maxAttempts = 30;
    let attempts = 0;
    
    const poll = async () => {
        try {
            const response = await fetch(`${API_BASE}/submission/${submissionId}`);
            const submission = await response.json();
            
            if (submission.status !== 'pending' && submission.status !== 'running') {
                alert(`Submission ${submissionId}: ${submission.status.toUpperCase()}\nScore: ${submission.score}\nRuntime: ${submission.runtime}ms`);
                if (currentContestId) {
                    loadLeaderboard(currentContestId);
                }
                return;
            }
            
            attempts++;
            if (attempts < maxAttempts) {
                setTimeout(poll, 2000);
            } else {
                alert('Submission is taking longer than expected. Please check later.');
            }
        } catch (error) {
            console.error('Error polling submission:', error);
        }
    };
    
    poll();
}

async function loadLeaderboard(contestId) {
    try {
        const response = await fetch(`${API_BASE}/leaderboard/${contestId}`);
        const leaderboard = await response.json();
        displayLeaderboard(leaderboard);
    } catch (error) {
        console.error('Error loading leaderboard:', error);
    }
}

function displayLeaderboard(leaderboard) {
    const container = document.getElementById('leaderboard');
    container.innerHTML = '<h4>Leaderboard</h4><div class="leaderboard">';
    
    if (leaderboard.length === 0) {
        container.innerHTML += '<p>No submissions yet</p></div>';
        return;
    }

    leaderboard.forEach((entry, index) => {
        const entryDiv = document.createElement('div');
        entryDiv.className = 'leaderboard-entry';
        entryDiv.innerHTML = `
            <strong>#${index + 1}</strong> ${entry.user_name}<br>
            Solved: ${entry.solved_count} | Penalty: ${entry.penalty} minutes
        `;
        container.querySelector('.leaderboard').appendChild(entryDiv);
    });
    
    container.innerHTML += '</div>';
}

// Admin functions
function showAdminTab(tabName, element) {
    document.getElementById('admin-contests-tab').style.display = tabName === 'contests' ? 'block' : 'none';
    document.getElementById('admin-problems-tab').style.display = tabName === 'problems' ? 'block' : 'none';
    document.getElementById('admin-testcases-tab').style.display = tabName === 'testcases' ? 'block' : 'none';
    
    // Update active tab
    document.querySelectorAll('.nav-link').forEach(link => link.classList.remove('active'));
    if (element) {
        element.classList.add('active');
    }
}

async function loadAdminContests() {
    try {
        const response = await fetch(`${API_BASE}/contests`);
        const contests = await response.json();
        displayAdminContests(contests);
    } catch (error) {
        console.error('Error loading contests:', error);
    }
}

function displayAdminContests(contests) {
    const container = document.getElementById('admin-contests-list');
    container.innerHTML = '<h4>Existing Contests</h4>';
    
    if (contests.length === 0) {
        container.innerHTML += '<p>No contests available</p>';
        return;
    }

    contests.forEach(contest => {
        const card = document.createElement('div');
        card.className = 'card mb-2';
        card.innerHTML = `
            <div class="card-body">
                <h5 class="card-title">${contest.title}</h5>
                <p class="card-text">
                    <small>ID: ${contest.id}</small><br>
                    Writer: ${contest.writer_name || 'N/A'}<br>
                    Start: ${new Date(contest.start_time).toLocaleString()}<br>
                    End: ${new Date(contest.end_time).toLocaleString()}<br>
                    Standings Frozen: ${contest.standings_frozen ? 'Yes' : 'No'}
                </p>
                <button class="btn btn-sm btn-primary edit-contest-btn" data-id="${contest.id}">Edit</button>
            </div>
        `;
        const editBtn = card.querySelector('.edit-contest-btn');
        editBtn.addEventListener('click', () => editContest(contest));
        container.appendChild(card);
    });
}

function editContest(contest) {
    document.getElementById('edit-contest-id').value = contest.id;
    document.getElementById('edit-contest-title').value = contest.title;
    document.getElementById('edit-contest-writer-name').value = contest.writer_name || '';
    
    // Format datetime-local format (YYYY-MM-DDTHH:mm)
    const startTime = new Date(contest.start_time);
    const endTime = new Date(contest.end_time);
    document.getElementById('edit-contest-start-time').value = startTime.toISOString().slice(0, 16);
    document.getElementById('edit-contest-end-time').value = endTime.toISOString().slice(0, 16);
    
    document.getElementById('edit-contest-standings-frozen').checked = contest.standings_frozen || false;
    
    if (contest.freeze_time) {
        const freezeTime = new Date(contest.freeze_time);
        document.getElementById('edit-contest-freeze-time').value = freezeTime.toISOString().slice(0, 16);
    }
    
    document.getElementById('freeze-time-container').style.display = contest.standings_frozen ? 'block' : 'none';
    document.getElementById('edit-contest-form-container').style.display = 'block';
    document.getElementById('edit-contest-form-container').scrollIntoView({ behavior: 'smooth' });
}

async function handleCreateContest(e) {
    e.preventDefault();
    const title = document.getElementById('contest-title-input').value;
    const writerName = document.getElementById('contest-writer-name').value;
    const startTime = document.getElementById('contest-start-time').value;
    const endTime = document.getElementById('contest-end-time').value;

    try {
        const response = await fetch(`${API_BASE}/contests`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${authToken}`
            },
            body: JSON.stringify({
                title: title,
                writer_name: writerName,
                start_time: new Date(startTime).toISOString(),
                end_time: new Date(endTime).toISOString()
            })
        });

        const data = await response.json();
        if (response.ok) {
            alert(`Contest created successfully! ID: ${data.id}`);
            document.getElementById('create-contest-form').reset();
            loadAdminContests();
            loadContests();
        } else {
            alert(data.error || 'Failed to create contest');
        }
    } catch (error) {
        alert('Error: ' + error.message);
    }
}

async function handleUpdateContest(e) {
    e.preventDefault();
    const contestId = document.getElementById('edit-contest-id').value;
    const title = document.getElementById('edit-contest-title').value;
    const writerName = document.getElementById('edit-contest-writer-name').value;
    const startTime = document.getElementById('edit-contest-start-time').value;
    const endTime = document.getElementById('edit-contest-end-time').value;
    const standingsFrozen = document.getElementById('edit-contest-standings-frozen').checked;
    const freezeTime = document.getElementById('edit-contest-freeze-time').value;

    const updateData = {
        title: title,
        writer_name: writerName,
        start_time: new Date(startTime).toISOString(),
        end_time: new Date(endTime).toISOString(),
        standings_frozen: standingsFrozen
    };

    if (freezeTime && standingsFrozen) {
        updateData.freeze_time = new Date(freezeTime).toISOString();
    }

    try {
        const response = await fetch(`${API_BASE}/contest/${contestId}`, {
            method: 'PUT',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${authToken}`
            },
            body: JSON.stringify(updateData)
        });

        const data = await response.json();
        if (response.ok) {
            alert('Contest updated successfully!');
            document.getElementById('edit-contest-form-container').style.display = 'none';
            loadAdminContests();
            loadContests();
        } else {
            alert(data.error || 'Failed to update contest');
        }
    } catch (error) {
        alert('Error: ' + error.message);
    }
}

async function handleCreateProblem(e) {
    e.preventDefault();
    const contestId = parseInt(document.getElementById('problem-contest-id').value);
    const title = document.getElementById('problem-title-input').value;
    const statement = document.getElementById('problem-statement').value;
    const timeLimit = parseInt(document.getElementById('problem-time-limit').value);
    const memoryLimit = parseInt(document.getElementById('problem-memory-limit').value);

    try {
        const response = await fetch(`${API_BASE}/problems`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${authToken}`
            },
            body: JSON.stringify({
                contest_id: contestId,
                title: title,
                statement: statement,
                time_limit: timeLimit,
                memory_limit: memoryLimit
            })
        });

        const data = await response.json();
        if (response.ok) {
            alert(`Problem created successfully! ID: ${data.id}`);
            document.getElementById('create-problem-form').reset();
        } else {
            alert(data.error || 'Failed to create problem');
        }
    } catch (error) {
        alert('Error: ' + error.message);
    }
}

async function handleCreateTestcase(e) {
    e.preventDefault();
    const problemId = parseInt(document.getElementById('testcase-problem-id').value);
    const input = document.getElementById('testcase-input').value;
    const expectedOutput = document.getElementById('testcase-expected-output').value;
    const isSample = document.getElementById('testcase-is-sample').checked;

    try {
        const response = await fetch(`${API_BASE}/testcases`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${authToken}`
            },
            body: JSON.stringify({
                problem_id: problemId,
                input: input,
                expected_output: expectedOutput,
                is_sample: isSample
            })
        });

        const data = await response.json();
        if (response.ok) {
            alert(`Testcase added successfully! ID: ${data.id}`);
            document.getElementById('create-testcase-form').reset();
        } else {
            alert(data.error || 'Failed to add testcase');
        }
    } catch (error) {
        alert('Error: ' + error.message);
    }
}

// Make functions available globally
window.viewProblem = viewProblem;
window.showSubmissionForm = showSubmissionForm;
window.showAdminTab = showAdminTab;

