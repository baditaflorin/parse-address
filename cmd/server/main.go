package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/parse-address/pkg/config"
	"github.com/parse-address/pkg/parser"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Starting address parser server on %s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Configuration: CORS=%v, RateLimit=%d/min, MaxInput=%d bytes",
		cfg.Security.EnableCORS, cfg.Security.RateLimitPerMin, cfg.Security.MaxInputLength)

	// Create parser instance
	p := parser.NewParser()

	// Setup router
	r := mux.NewRouter()

	// API routes
	api := r.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/parse", parseHandler(p)).Methods("POST", "OPTIONS")
	api.HandleFunc("/health", healthHandler).Methods("GET")
	api.HandleFunc("/config", configHandler(cfg)).Methods("GET")

	// Static file server for GUI
	r.HandleFunc("/", indexHandler).Methods("GET")
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	// Middleware
	handler := loggingMiddleware(r)
	handler = corsMiddleware(cfg, handler)
	handler = securityHeadersMiddleware(handler)
	handler = requestSizeLimitMiddleware(cfg.Server.MaxRequestSize, handler)

	// Create server
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server listening on http://%s:%d", cfg.Server.Host, cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

// Handlers

type parseRequest struct {
	Address string `json:"address"`
	Type    string `json:"type,omitempty"` // "standard", "informal", "intersection", "po_box", "auto"
}

type parseResponse struct {
	Success bool                `json:"success"`
	Error   string              `json:"error,omitempty"`
	Result  *parser.ParseResult `json:"result,omitempty"`
}

func parseHandler(p *parser.Parser) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req parseRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, parseResponse{
				Success: false,
				Error:   "Invalid request format",
			})
			return
		}

		if req.Address == "" {
			respondJSON(w, http.StatusBadRequest, parseResponse{
				Success: false,
				Error:   "Address field is required",
			})
			return
		}

		var result *parser.ParseResult
		var err error

		// Route to appropriate parser based on type
		switch req.Type {
		case "standard":
			addr := p.ParseAddress(req.Address)
			result = &parser.ParseResult{Type: "address", Address: addr}
		case "informal":
			addr := p.ParseInformalAddress(req.Address)
			result = &parser.ParseResult{Type: "address", Address: addr}
		case "intersection":
			inter := p.ParseIntersection(req.Address)
			result = &parser.ParseResult{Type: "intersection", Intersection: inter}
		case "po_box":
			addr := p.ParsePoAddress(req.Address)
			result = &parser.ParseResult{Type: "po_box", Address: addr}
		default: // "auto" or empty
			result, err = p.ParseLocation(req.Address)
		}

		if err != nil {
			respondJSON(w, http.StatusBadRequest, parseResponse{
				Success: false,
				Error:   fmt.Sprintf("Parse error: %v", err),
			})
			return
		}

		respondJSON(w, http.StatusOK, parseResponse{
			Success: true,
			Result:  result,
		})
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func configHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Return safe subset of config (no sensitive data)
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"maxInputLength": cfg.Security.MaxInputLength,
			"corsEnabled":    cfg.Security.EnableCORS,
		})
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(indexHTML))
}

// Middleware

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("%s %s %s", r.Method, r.RequestURI, r.RemoteAddr)
		next.ServeHTTP(w, r)
		log.Printf("%s %s - completed in %v", r.Method, r.RequestURI, time.Since(start))
	})
}

func corsMiddleware(cfg *config.Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cfg.Security.EnableCORS {
			origin := r.Header.Get("Origin")
			if origin == "" {
				origin = "*"
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "3600")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")
		next.ServeHTTP(w, r)
	})
}

func requestSizeLimitMiddleware(maxSize int64, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxSize)
		next.ServeHTTP(w, r)
	})
}

// Utilities

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// Embedded HTML for the web interface
const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>US Address Parser - Testing Interface</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            padding: 20px;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            border-radius: 12px;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            overflow: hidden;
        }
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 30px;
            text-align: center;
        }
        .header h1 { font-size: 32px; margin-bottom: 10px; }
        .header p { opacity: 0.9; }
        .main { padding: 30px; }
        .input-section {
            background: #f7fafc;
            padding: 25px;
            border-radius: 8px;
            margin-bottom: 25px;
        }
        label {
            display: block;
            font-weight: 600;
            margin-bottom: 8px;
            color: #2d3748;
        }
        textarea, select, input {
            width: 100%;
            padding: 12px;
            border: 2px solid #e2e8f0;
            border-radius: 6px;
            font-size: 16px;
            font-family: inherit;
            transition: border-color 0.2s;
        }
        textarea:focus, select:focus, input:focus {
            outline: none;
            border-color: #667eea;
        }
        textarea { min-height: 100px; resize: vertical; }
        .button-group {
            display: flex;
            gap: 10px;
            margin-top: 15px;
        }
        button {
            flex: 1;
            padding: 14px 24px;
            border: none;
            border-radius: 6px;
            font-size: 16px;
            font-weight: 600;
            cursor: pointer;
            transition: all 0.2s;
        }
        .btn-primary {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
        }
        .btn-primary:hover { transform: translateY(-2px); box-shadow: 0 4px 12px rgba(102, 126, 234, 0.4); }
        .btn-secondary {
            background: #e2e8f0;
            color: #2d3748;
        }
        .btn-secondary:hover { background: #cbd5e0; }
        .results {
            background: #f7fafc;
            border-radius: 8px;
            padding: 20px;
            margin-top: 20px;
        }
        .results h2 {
            color: #2d3748;
            margin-bottom: 15px;
            font-size: 20px;
        }
        .result-card {
            background: white;
            border-radius: 6px;
            padding: 15px;
            margin-bottom: 10px;
            border-left: 4px solid #667eea;
        }
        .result-item {
            display: flex;
            padding: 8px 0;
            border-bottom: 1px solid #e2e8f0;
        }
        .result-item:last-child { border-bottom: none; }
        .result-label {
            font-weight: 600;
            color: #4a5568;
            width: 150px;
        }
        .result-value {
            color: #2d3748;
            flex: 1;
        }
        .examples {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 15px;
            margin-top: 20px;
        }
        .example-card {
            background: white;
            padding: 15px;
            border-radius: 6px;
            border: 2px solid #e2e8f0;
            cursor: pointer;
            transition: all 0.2s;
        }
        .example-card:hover {
            border-color: #667eea;
            transform: translateY(-2px);
            box-shadow: 0 4px 12px rgba(0,0,0,0.1);
        }
        .example-title {
            font-weight: 600;
            color: #667eea;
            margin-bottom: 5px;
        }
        .example-text {
            color: #4a5568;
            font-size: 14px;
        }
        .error {
            background: #fed7d7;
            color: #9b2c2c;
            padding: 15px;
            border-radius: 6px;
            margin-top: 15px;
        }
        .success {
            background: #c6f6d5;
            color: #22543d;
            padding: 15px;
            border-radius: 6px;
            margin-top: 15px;
        }
        .badge {
            display: inline-block;
            padding: 4px 12px;
            border-radius: 12px;
            font-size: 12px;
            font-weight: 600;
            text-transform: uppercase;
        }
        .badge-address { background: #bee3f8; color: #2c5282; }
        .badge-intersection { background: #fbd38d; color: #7c2d12; }
        .badge-po { background: #fbb6ce; color: #702459; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🏠 US Address Parser</h1>
            <p>Secure, validated address parsing with live testing interface</p>
        </div>
        <div class="main">
            <div class="input-section">
                <label for="address">Enter Address</label>
                <textarea id="address" placeholder="1005 N Gravenstein Highway Sebastopol CA 95472"></textarea>

                <label for="parseType" style="margin-top: 15px;">Parse Type</label>
                <select id="parseType">
                    <option value="auto">Auto-detect</option>
                    <option value="standard">Standard Address</option>
                    <option value="informal">Informal Address</option>
                    <option value="intersection">Intersection</option>
                    <option value="po_box">PO Box</option>
                </select>

                <div class="button-group">
                    <button class="btn-primary" onclick="parseAddress()">Parse Address</button>
                    <button class="btn-secondary" onclick="clearResults()">Clear</button>
                </div>
            </div>

            <div id="results"></div>

            <div class="results">
                <h2>Example Addresses (click to test)</h2>
                <div class="examples">
                    <div class="example-card" onclick="setAddress('1005 N Gravenstein Highway Sebastopol CA 95472')">
                        <div class="example-title">Standard Address</div>
                        <div class="example-text">1005 N Gravenstein Highway Sebastopol CA 95472</div>
                    </div>
                    <div class="example-card" onclick="setAddress('123 Main St Apt 4B San Francisco, CA 94105')">
                        <div class="example-title">With Unit Number</div>
                        <div class="example-text">123 Main St Apt 4B San Francisco, CA 94105</div>
                    </div>
                    <div class="example-card" onclick="setAddress('Mission St and Valencia St, San Francisco CA')">
                        <div class="example-title">Intersection</div>
                        <div class="example-text">Mission St and Valencia St, San Francisco CA</div>
                    </div>
                    <div class="example-card" onclick="setAddress('PO Box 1234 New York NY 10001')">
                        <div class="example-title">PO Box</div>
                        <div class="example-text">PO Box 1234 New York NY 10001</div>
                    </div>
                </div>
            </div>
        </div>
    </div>

    <script>
        function setAddress(addr) {
            document.getElementById('address').value = addr;
            parseAddress();
        }

        function clearResults() {
            document.getElementById('address').value = '';
            document.getElementById('results').innerHTML = '';
        }

        async function parseAddress() {
            const address = document.getElementById('address').value.trim();
            const parseType = document.getElementById('parseType').value;
            const resultsDiv = document.getElementById('results');

            if (!address) {
                resultsDiv.innerHTML = '<div class="error">Please enter an address</div>';
                return;
            }

            resultsDiv.innerHTML = '<div class="success">Parsing...</div>';

            try {
                const response = await fetch('/api/v1/parse', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ address, type: parseType })
                });

                const data = await response.json();

                if (!data.success) {
                    resultsDiv.innerHTML = '<div class="error">Error: ' + (data.error || 'Unknown error') + '</div>';
                    return;
                }

                displayResults(data.result);
            } catch (error) {
                resultsDiv.innerHTML = '<div class="error">Network error: ' + error.message + '</div>';
            }
        }

        function displayResults(result) {
            const resultsDiv = document.getElementById('results');
            let html = '<div class="results"><h2>Parse Results</h2>';

            if (result.type === 'none') {
                html += '<div class="error">Could not parse address</div>';
            } else if (result.type === 'intersection' && result.intersection) {
                html += '<span class="badge badge-intersection">Intersection</span>';
                html += '<div class="result-card">';
                const inter = result.intersection;
                html += formatResultItem('Street 1', [inter.prefix1, inter.street1, inter.type1, inter.suffix1].filter(Boolean).join(' '));
                html += formatResultItem('Street 2', [inter.prefix2, inter.street2, inter.type2, inter.suffix2].filter(Boolean).join(' '));
                if (inter.city) html += formatResultItem('City', inter.city);
                if (inter.state) html += formatResultItem('State', inter.state);
                if (inter.zip) html += formatResultItem('ZIP', inter.zip);
                html += '</div>';
            } else if (result.address) {
                const badge = result.type === 'po_box' ? 'badge-po' : 'badge-address';
                const label = result.type === 'po_box' ? 'PO Box' : 'Address';
                html += '<span class="badge ' + badge + '">' + label + '</span>';
                html += '<div class="result-card">';
                const addr = result.address;
                if (addr.number) html += formatResultItem('Number', addr.number);
                if (addr.prefix) html += formatResultItem('Prefix', addr.prefix);
                if (addr.street) html += formatResultItem('Street', addr.street);
                if (addr.type) html += formatResultItem('Type', addr.type);
                if (addr.suffix) html += formatResultItem('Suffix', addr.suffix);
                if (addr.sec_unit_type) html += formatResultItem('Unit Type', addr.sec_unit_type);
                if (addr.sec_unit_num) html += formatResultItem('Unit #', addr.sec_unit_num);
                if (addr.city) html += formatResultItem('City', addr.city);
                if (addr.state) html += formatResultItem('State', addr.state);
                if (addr.zip) html += formatResultItem('ZIP', addr.zip);
                if (addr.plus4) html += formatResultItem('ZIP+4', addr.plus4);
                html += '</div>';
            }

            html += '</div>';
            resultsDiv.innerHTML = html;
        }

        function formatResultItem(label, value) {
            if (!value) return '';
            return '<div class="result-item"><div class="result-label">' + label + ':</div><div class="result-value">' + value + '</div></div>';
        }

        // Allow Enter key to submit
        document.getElementById('address').addEventListener('keypress', function(e) {
            if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                parseAddress();
            }
        });
    </script>
</body>
</html>`
