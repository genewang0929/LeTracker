# LeTracker - Intelligent LeetCode SRS Scheduler

**LeTracker** is a LeetCode practice assistant tool based on a **Spaced Repetition System (SRS)**. Unlike traditional practice lists (e.g., NeetCode 150) that enforce sequential order, LeTracker tracks your historical submission performance to automatically recommend the top 3 questions you are "about to forget" or "most need to practice" each day.

## Key Features

* **🧠 Smart Review Algorithm**: Utilizes a modified **SM-2 Algorithm** optimized for coding problems (e.g., penalty mechanisms for "Hard" ratings, retention bonuses for long-term memory recall).
* **🔄 History Replay**: Upon initialization, the system automatically imports your LeetCode submission history and reconstructs your current mastery state via a "Time-Travel Simulation," rather than starting from zero.
* **📊 Automated Tracking**: Integrates with a Chrome Extension to automatically record your submission results, eliminating the need for manual data entry.
* **🎯 Daily Tasks**: Recommends top priority questions daily based on **Relative Overdue** sorting.

## Tech Stack

* **Language**: Go (Golang)
* **Web Framework**: Gin
* **Database**: PostgreSQL (via Supabase)
* **Architecture**: Clean Architecture (Layered: Handler -> Service -> Repository)

## Project Structure

```text
letracker/
├── cmd/api/            # Application entry point (main.go)
├── internal/
│   ├── entity/         # Data model definitions (DTOs/Entities)
│   ├── repository/     # Data Access Layer (SQL implementations)
│   ├── service/        # Business Logic (Import, Replay Logic)
│   └── handler/        # HTTP Request Handlers (API Endpoints)
├── pkg/srs/            # Core SRS Algorithm Engine (Pure Math)
└── extension/          # Chrome Extension (Frontend Data Fetching)
```

## Getting Started

### 1. Prerequisites
* **Go**: Version 1.20 or higher
* **Supabase**: A project with a PostgreSQL database
* **Git**: Version control system

### 2. Installation & Configuration

Clone the repository:

```bash
git clone [https://github.com/YOUR_USERNAME/letracker.git](https://github.com/YOUR_USERNAME/letracker.git)
cd letracker
```

Set up environment variables:

Create a `.env` file in the root directory and add your Supabase connection string.

```env
# Enter your Supabase connection string in the .env file
# Note: If using Supabase Transaction/Session Pooler, ensure you use port 6543 and the correct user project ID.
DB_DSN="postgres://postgres.your-project-id:password@aws-0-pooler.supabase.com:6543/postgres?sslmode=disable"
```

### 3. Installation & Configuration
Execute the following SQL in your Supabase SQL Editor to create the necessary tables:
```sql
-- 1. Questions Table
CREATE TABLE questions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    leetcode_frontend_id INTEGER,
    title TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    difficulty TEXT,
    is_neetcode_150 BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 2. Study Logs (Immutable History)
CREATE TABLE study_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL, -- Can link to auth.users
    question_id UUID NOT NULL REFERENCES questions(id),
    status TEXT,
    mastery_level SMALLINT,
    attempted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 3. User Question Stats (SRS State)
CREATE TABLE user_question_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    question_id UUID NOT NULL REFERENCES questions(id),
    streak INTEGER DEFAULT 0,
    ease_factor FLOAT DEFAULT 2.5,
    interval_days INTEGER DEFAULT 0,
    status TEXT DEFAULT 'NEW',
    next_review_at TIMESTAMP WITH TIME ZONE,
    last_reviewed_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(user_id, question_id)
);
```

### 4. Running the Server
```bash
# Load .env variables and run the application
export $(cat .env | xargs) && go run cmd/api/main.go
```
You should see Server starting on port 8080... indicating the server is running.

### 5. API Endpoints
* `POST /api/v1/history`: Import LeetCode submission history (JSON format).
* `GET /api/v1/tasks`: Retrieve today's recommended tasks.
* `POST /api/v1/submit`: Submit a review result for a single question.
