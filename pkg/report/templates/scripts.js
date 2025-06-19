// Chart.js configuration for PI Scanner reports

// Risk Distribution Chart
function createRiskDistributionChart(data) {
    const ctx = document.getElementById('riskChart');
    if (!ctx) return;
    
    const riskColors = {
        'CRITICAL': '#dc3545',
        'HIGH': '#fd7e14',
        'MEDIUM': '#ffc107',
        'LOW': '#28a745'
    };
    
    const chartData = {
        labels: ['Critical', 'High', 'Medium', 'Low'],
        datasets: [{
            data: [
                data.Summary.CriticalCount || 0,
                data.Summary.HighCount || 0,
                data.Summary.MediumCount || 0,
                data.Summary.LowCount || 0
            ],
            backgroundColor: [
                riskColors.CRITICAL,
                riskColors.HIGH,
                riskColors.MEDIUM,
                riskColors.LOW
            ],
            borderWidth: 2,
            borderColor: '#fff'
        }]
    };
    
    new Chart(ctx, {
        type: 'doughnut',
        data: chartData,
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: {
                    position: 'bottom',
                    labels: {
                        padding: 15,
                        font: {
                            size: 12
                        }
                    }
                },
                tooltip: {
                    callbacks: {
                        label: function(context) {
                            const label = context.label || '';
                            const value = context.parsed || 0;
                            const total = context.dataset.data.reduce((a, b) => a + b, 0);
                            const percentage = total > 0 ? ((value / total) * 100).toFixed(1) : '0';
                            return `${label}: ${value} (${percentage}%)`;
                        }
                    }
                }
            }
        }
    });
}

// PI Type Distribution Chart
function createPITypeDistributionChart(data) {
    const ctx = document.getElementById('typeChart');
    if (!ctx) return;
    
    if (!data.Statistics || !data.Statistics.TypeDistribution) {
        console.warn('No type distribution data available');
        return;
    }
    
    const typeColors = {
        'TFN': '#1e3c72',
        'MEDICARE': '#2a5298',
        'ABN': '#3498db',
        'BSB': '#5dade2',
        'CREDIT_CARD': '#85c1e2',
        'EMAIL': '#aed6f1',
        'PHONE': '#d6eaf8',
        'NAME': '#ebf5fb',
        'ADDRESS': '#f4f6f7',
        'DEFAULT': '#95a5a6'
    };
    
    const sortedTypes = Object.entries(data.Statistics.TypeDistribution)
        .sort((a, b) => b[1] - a[1])
        .slice(0, 8); // Top 8 types
    
    const chartData = {
        labels: sortedTypes.map(([type]) => formatPIType(type)),
        datasets: [{
            data: sortedTypes.map(([, count]) => count),
            backgroundColor: sortedTypes.map(([type]) => typeColors[type] || typeColors.DEFAULT),
            borderWidth: 2,
            borderColor: '#fff'
        }]
    };
    
    new Chart(ctx, {
        type: 'bar',
        data: chartData,
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: {
                    display: false
                },
                tooltip: {
                    callbacks: {
                        label: function(context) {
                            const value = context.parsed.y || 0;
                            const total = data.Summary.TotalFindings || 0;
                            const percentage = total > 0 ? ((value / total) * 100).toFixed(1) : '0';
                            return `Count: ${value} (${percentage}%)`;
                        }
                    }
                }
            },
            scales: {
                y: {
                    beginAtZero: true,
                    ticks: {
                        stepSize: 1
                    },
                    title: {
                        display: true,
                        text: 'Number of Findings'
                    }
                },
                x: {
                    title: {
                        display: true,
                        text: 'PI Type'
                    }
                }
            }
        }
    });
}

// Format PI type for display
function formatPIType(type) {
    const typeMap = {
        'TFN': 'Tax File Number',
        'MEDICARE': 'Medicare',
        'ABN': 'ABN',
        'BSB': 'BSB',
        'CREDIT_CARD': 'Credit Card',
        'EMAIL': 'Email',
        'PHONE': 'Phone',
        'NAME': 'Name',
        'ADDRESS': 'Address',
        'PASSPORT': 'Passport',
        'DRIVER_LICENSE': 'Driver License',
        'IP': 'IP Address'
    };
    return typeMap[type] || type;
}

// Initialize charts when DOM is loaded
document.addEventListener('DOMContentLoaded', function() {
    // Get the data from the template (this will be injected by the Go template)
    const reportData = window.reportData || {
        Summary: {
            CriticalCount: {{.Summary.CriticalCount}},
            HighCount: {{.Summary.HighCount}},
            MediumCount: {{.Summary.MediumCount}},
            LowCount: {{.Summary.LowCount}},
            TotalFindings: {{.Summary.TotalFindings}}
        },
        Statistics: {
            TypeDistribution: {{.Statistics.TypeDistribution | jsonify}}
        }
    };
    
    // Create charts
    createRiskDistributionChart(reportData);
    createPITypeDistributionChart(reportData);
    
    // Add print functionality
    const printButton = document.getElementById('printReport');
    if (printButton) {
        printButton.addEventListener('click', function() {
            window.print();
        });
    }
    
    // Add export functionality
    const exportButton = document.getElementById('exportData');
    if (exportButton) {
        exportButton.addEventListener('click', function() {
            exportReportData(reportData);
        });
    }
    
    // Smooth scroll for anchor links
    document.querySelectorAll('a[href^="#"]').forEach(anchor => {
        anchor.addEventListener('click', function(e) {
            e.preventDefault();
            const target = document.querySelector(this.getAttribute('href'));
            if (target) {
                target.scrollIntoView({
                    behavior: 'smooth',
                    block: 'start'
                });
            }
        });
    });
    
    // Expandable finding details
    document.querySelectorAll('.finding-card').forEach(card => {
        card.addEventListener('click', function() {
            this.classList.toggle('expanded');
        });
    });
});

// Export report data as JSON
function exportReportData(data) {
    const jsonStr = JSON.stringify(data, null, 2);
    const blob = new Blob([jsonStr], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `pi-scanner-report-${data.ReportID}.json`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
}

// Helper function to format dates
function formatDate(dateStr) {
    const date = new Date(dateStr);
    return date.toLocaleDateString('en-AU', {
        year: 'numeric',
        month: 'short',
        day: 'numeric'
    });
}

// Helper function to format times
function formatTime(dateStr) {
    const date = new Date(dateStr);
    return date.toLocaleString('en-AU', {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
        timeZoneName: 'short'
    });
}

// Add dynamic filtering for findings
function initializeFilters() {
    const filterButtons = document.querySelectorAll('.filter-button');
    const findingCards = document.querySelectorAll('.finding-card');
    
    filterButtons.forEach(button => {
        button.addEventListener('click', function() {
            const filterType = this.dataset.filter;
            
            // Update active state
            filterButtons.forEach(btn => btn.classList.remove('active'));
            this.classList.add('active');
            
            // Filter findings
            findingCards.forEach(card => {
                if (filterType === 'all' || card.dataset.type === filterType) {
                    card.style.display = 'block';
                } else {
                    card.style.display = 'none';
                }
            });
        });
    });
}

// Initialize filters if present
if (document.querySelector('.filter-button')) {
    initializeFilters();
}

// Add search functionality for findings
function initializeSearch() {
    const searchInput = document.getElementById('findingSearch');
    if (!searchInput) return;
    
    const findingCards = document.querySelectorAll('.finding-card');
    
    searchInput.addEventListener('input', function() {
        const searchTerm = this.value.toLowerCase();
        
        findingCards.forEach(card => {
            const text = card.textContent.toLowerCase();
            if (text.includes(searchTerm)) {
                card.style.display = 'block';
            } else {
                card.style.display = 'none';
            }
        });
    });
}

// Initialize search if present
initializeSearch();

// Add keyboard shortcuts
document.addEventListener('keydown', function(e) {
    // Ctrl/Cmd + P for print
    if ((e.ctrlKey || e.metaKey) && e.key === 'p') {
        e.preventDefault();
        window.print();
    }
    
    // Ctrl/Cmd + S for export
    if ((e.ctrlKey || e.metaKey) && e.key === 's') {
        e.preventDefault();
        const exportButton = document.getElementById('exportData');
        if (exportButton) {
            exportButton.click();
        }
    }
});

// Add responsive table handling
function initializeResponsiveTables() {
    const tables = document.querySelectorAll('table');
    
    tables.forEach(table => {
        const wrapper = document.createElement('div');
        wrapper.className = 'table-responsive';
        table.parentNode.insertBefore(wrapper, table);
        wrapper.appendChild(table);
    });
}

// Initialize responsive tables
initializeResponsiveTables();

// Progress indicator for long reports
function showProgress() {
    const progressBar = document.createElement('div');
    progressBar.className = 'progress-bar';
    progressBar.innerHTML = '<div class="progress-fill"></div>';
    document.body.appendChild(progressBar);
    
    window.addEventListener('scroll', function() {
        const scrolled = window.scrollY;
        const height = document.documentElement.scrollHeight - window.innerHeight;
        const progress = (scrolled / height) * 100;
        
        const fill = progressBar.querySelector('.progress-fill');
        fill.style.width = progress + '%';
    });
}

// Show progress for long reports
if (document.body.scrollHeight > window.innerHeight * 1.5) {
    showProgress();
}