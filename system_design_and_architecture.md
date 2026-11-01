# Aegis — System Design, Architecture & Entity Specifications

A complete granular reference of the **Aegis AI Gateway** architecture designed for backend engineering and distributed systems interview preparation.

---

## 1. Entity & Attribute Catalog (Isolated Entities)

Every core domain object, database table, cache payload, and in-memory data structure used across Aegis:

```mermaid
classDiagram
    class ChatRequest {
        +string model
        +Message[] messages
        +bool stream
    }

    class Message {
        +string role
        +string content
    }

    class ChatResponse {
        +string id
        +string object
        +int64 created
        +string model
        +Choice[] choices
    }

    class Choice {
        +int index
        +Message message
        +string finish_reason
    }

    class ExactCacheEntry_Redis {
        +string key "exact:sha256(canonical_payload)"
        +string value "json(ChatResponse)"
        +int ttl_seconds "86400 (24h)"
    }

    class SemanticCacheRecord_PostgreSQL {
        +int64 id "PK (BIGSERIAL)"
        +string prompt "TEXT"
        +string model "VARCHAR(100)"
        +vector_768 embedding "HNSW Cosine Index"
        +jsonb response "ChatResponse JSON"
        +timestamptz created_at "TIMESTAMPTZ"
    }

    class TokenBucketRecord_Redis {
        +string key "ratelimit:client_ip"
        +float64 tokens
        +int64 last_refill_timestamp
    }

    class ModelRoutingDecision {
        +string class "FAST | BALANCED | DEEP"
        +string selected_model
        +map semantic_scores
        +map heuristic_scores
        +map final_scores
        +string reason
    }

    class FailoverTrackingInfo {
        +bool triggered
        +error primary_error
        +int attempt_count
        +string active_provider "primary | secondary"
    }

    class PrometheusMetricEntry {
        +string metric_name
        +map labels "route, method, status, cache, layer"
        +float64 value
        +float64 duration_bucket
    }
```

---

## 2. Entity-Relationship (ER) Diagram

Data model relationships, foreign keys, cardinality mappings, and entity associations:

```mermaid
erDiagram
    CHAT_REQUEST ||--|{ MESSAGE : "contains (1..N)"
    CHAT_REQUEST ||--o| MODEL_ROUTING_DECISION : "determines (1..1)"
    CHAT_REQUEST ||--o| EXACT_CACHE_ENTRY : "hashes to (1..1)"
    CHAT_REQUEST ||--o| SEMANTIC_CACHE_RECORD : "matches vector (0..1)"

    CHAT_REQUEST ||--|| CHAT_RESPONSE : "generates (1..1)"
    CHAT_RESPONSE ||--|{ CHOICE : "contains (1..N)"
    CHOICE ||--|| MESSAGE : "encapsulates (1..1)"

    CLIENT_IP ||--|| TOKEN_BUCKET : "owns rate bucket (1..1)"
    CLIENT_IP ||--o{ CHAT_REQUEST : "originates (1..N)"

    CHAT_RESPONSE ||--o| EXACT_CACHE_ENTRY : "cached as value (1..1)"
    CHAT_RESPONSE ||--o| SEMANTIC_CACHE_RECORD : "persisted as JSONB (1..1)"

    PROVIDER_FAILOVER ||--|| CHAT_RESPONSE : "tracks execution path (1..1)"

    CHAT_REQUEST {
        string model
        bool stream
    }

    MESSAGE {
        string role
        string content
    }

    CHAT_RESPONSE {
        string id PK
        string object
        int64 created
        string model
    }

    CHOICE {
        int index
        string finish_reason
    }

    CLIENT_IP {
        string ip_address PK
    }

    TOKEN_BUCKET {
        string key PK "ratelimit:ip"
        float64 tokens
        int64 last_refill
    }

    EXACT_CACHE_ENTRY {
        string key PK "exact:sha256"
        string payload_json
        int ttl
    }

    SEMANTIC_CACHE_RECORD {
        bigserial id PK
        string prompt
        string model
        vector embedding "vector(768)"
        jsonb response
        timestamptz created_at
    }

    MODEL_ROUTING_DECISION {
        string route_class "FAST, BALANCED, DEEP"
        string selected_model
        float64 fast_score
        float64 balanced_score
        float64 deep_score
    }

    PROVIDER_FAILOVER {
        bool triggered
        string active_provider
        int attempts
    }
```

---

## 3. Data Flow Diagram (DFD) — Level 0: Context Diagram

System-level view treating Aegis as a single process with external boundaries:

```mermaid
flowchart LR
    Client["Client Application / Developer"]
    Aegis(("0.0<br/>Aegis AI Gateway<br/>(Go Reverse Proxy)"))
    PrimaryLLM["Primary LLM Provider<br/>(Google Gemini API)"]
    SecondaryLLM["Secondary LLM Provider<br/>(Fallback Gemini)"]
    Prometheus["Prometheus Monitoring Server"]

    Client -->|"1. POST /v1/chat/completions (Prompt Payload)"| Aegis
    Aegis -->|"2. HTTP 200 JSON Response / SSE Stream"| Client
    Aegis -->|"3. HTTP 429 Too Many Requests (Rate Limited)"| Client

    Aegis -->|"4. LLM Generation Request (Cold / Miss)"| PrimaryLLM
    PrimaryLLM -->|"5. Generated Text Completion"| Aegis
    PrimaryLLM -.->|"5a. 503 / 429 Retryable Error"| Aegis

    Aegis -->|"6. Failover LLM Request (If Primary Fails)"| SecondaryLLM
    SecondaryLLM -->|"7. Fallback Completion"| Aegis

    Prometheus -->|"8. GET /metrics (5s Scrape Interval)"| Aegis
    Aegis -->|"9. Prometheus Metrics & Latency Histograms"| Prometheus
```

---

## 4. Data Flow Diagram (DFD) — Level 1: Subsystem Decomposition

Granular data flow across internal gateway components and physical persistence stores:

```mermaid
flowchart TD
    Client["Client Application"]
    
    subgraph DataStores ["Data Stores"]
        D1[("D1: Redis Token Buckets")]
        D2[("D2: Redis Exact Cache (SHA-256)")]
        D3[("D3: PostgreSQL pgvector (HNSW Index)")]
        D4[("D4: Prometheus Metric Registry")]
    end

    subgraph Pipeline ["Aegis Gateway Execution Pipeline"]
        P1("1.0 Rate Limiting<br/>(Atomic Token Bucket)")
        P2("2.0 Exact Cache Check<br/>(SHA-256 Hash Lookup)")
        P3("3.0 Semantic Cache Check<br/>(Embed + pgvector Cosine Search)")
        P4("4.0 Hybrid Model Router<br/>(Semantic Prototypes + Heuristics)")
        P5("5.0 Upstream Execution & Failover<br/>(Primary -> Error Classify -> Secondary)")
        P6("6.0 Cache Invalidation & Backfill<br/>(Store pgvector + Backfill Redis)")
        P7("7.0 Metrics & Telemetry<br/>(Middleware Duration Histogram)")
    end

    UpstreamPrimary["Primary LLM (Gemini)"]
    UpstreamSecondary["Secondary LLM (Failover)"]

    %% Flow 1: Ingress & Rate Limiting
    Client -->|"Inbound HTTP Request"| P1
    P1 <-->|"Atomic Lua: Eval Capacity & Refill"| D1
    P1 -->|"If tokens < 1: HTTP 429"| Client
    P1 -->|"If allowed: Forward Request"| P2

    %% Flow 2: Exact Cache
    P2 <-->|"Check exact:sha256 key"| D2
    P2 -->|"EXACT_HIT (2.1ms): Return Cached JSON"| P7

    %% Flow 3: Semantic Cache
    P2 -->|"EXACT_MISS: Forward"| P3
    P3 <-->|"FindSimilar: Cosine Distance <= 0.20"| D3
    P3 -->|"SEMANTIC_HIT (678ms): Return JSON"| P6
    P6 -->|"Backfill Redis exact:sha256 key"| D2
    P6 -->|"Forward response"| P7

    %% Flow 4: Model Routing
    P3 -->|"SEMANTIC_MISS: Forward"| P4
    P4 -->|"Classified Model (FAST/BALANCED/DEEP)"| P5

    %% Flow 5: Upstream LLM & Failover
    P5 -->|"Attempt Primary Call"| UpstreamPrimary
    UpstreamPrimary -->|"Success Response"| P6
    UpstreamPrimary -.->|"Retryable 5xx/429 Error"| P5
    P5 -.->|"Trigger Failover"| UpstreamSecondary
    UpstreamSecondary -.->|"Failover Response"| P6

    %% Storage on Cold Miss
    P6 -->|"Insert Vector & Response (JSONB)"| D3
    P6 -->|"Set Redis Exact Key (24h TTL)"| D2
    P6 -->|"Return Fresh Upstream Response"| P7

    %% Flow 6: Observability Egress
    P7 -->|"Increment Counters & Record Latency"| D4
    P7 -->|"Deliver HTTP 200 / SSE Stream to Client"| Client
```

---

## 5. System Design Interview Cheatsheet

### A. The Two-Tier Caching Rationale
* **Layer 1: Redis Exact Cache**
  * Time Complexity: $\mathcal{O}(1)$.
  * Latency: **~1.6–2.1 ms**.
  * Throughput: **1,417+ req/s** sustained across 100 concurrent VUs.
  * Handles identical queries with zero embedding or GPU cost.
* **Layer 2: PostgreSQL `pgvector` Semantic Cache**
  * Time Complexity: $\mathcal{O}(\log N)$ via HNSW vector index.
  * Latency: **~678 ms** (vs. 9,110 ms cold upstream, yielding a **92.55% latency reduction**).
  * Catches paraphrased sentences, typos, and semantic equivalents.
* **The Backfill Optimization**:
  * On a semantic hit, Aegis automatically backfills Redis exact cache for that specific paraphrase.
  * The second time that paraphrase is called, latency drops from **678 ms $\rightarrow$ 2.1 ms** (a **322× acceleration**).

### B. Distributed Token Bucket via Redis Lua
* **Problem**: In distributed systems, separate read-modify-write calls cause race conditions under high concurrency (Check-Then-Act anti-pattern).
* **Solution**: Executing the token bucket calculation inside an atomic Redis Lua script ensures zero lock contention.
* **Empirical Validation**: Sustained **154,378 requests at 7,525.8 req/s** with zero 5xx server crashes and 2.07 ms rejection latency.

### C. The Real Latency Bottleneck: Vector DB vs. Cloud Embedding
* In our benchmarks, we instrumented `X-Vector-Duration-Ms` and `X-Embedding-Duration-Ms`:
  * Local PostgreSQL `pgvector` HNSW search completed in **1.01 ms**.
  * The Google Gemini Cloud embedding network round-trip took **684.54 ms**.
  * **Interview Takeaway**: The vector database is **not** the bottleneck (it accounts for $<0.2\%$ of retrieval time). The external HTTPS call to Google dominates. In ultra-low-latency setups, hosting an in-cluster embedding container (ONNX / TEI) drops semantic retrieval under 15 ms.

### D. Production Traffic Reality (Mixed Traffic Test)
* In real-world production traffic (64% exact hits, 25% semantic hits, 11% cold misses):
  * **Overall Cache Hit Ratio**: **89.01%** (cutting upstream LLM token costs by ~89%).
  * **System Median (P50) Latency**: **1.57 ms** (even with 11% of requests taking 15s+ upstream, 50% of all users experience sub-2ms response times).
