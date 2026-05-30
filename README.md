# 💰 Bill-Splitting Application

A full-stack web application for managing shared expenses and settling bills within groups. Users can create groups, add members, track expenses, calculate splits, and manage settlements.

---

## 📋 Table of Contents

- [Project Overview](#project-overview)
- [Technology Stack](#technology-stack)
- [Project Structure](#project-structure)
- [Architecture & Pipeline](#architecture--pipeline)
- [Database Schema](#database-schema)
- [Setup & Installation](#setup--installation)
- [API Endpoints](#api-endpoints)
- [Development Workflow](#development-workflow)
- [Contributing](#contributing)

---

## 🎯 Project Overview

Bill-Splitting is a collaborative expense management application designed to simplify shared finances. Whether splitting rent, group dinners, or vacation costs, this application:

- ✅ Allows users to create groups and invite members (no registration required for members)
- ✅ Tracks shared expenses and calculates fair splits
- ✅ Automatically computes who owes whom and settlement recommendations
- ✅ Provides a clean, intuitive dashboard for managing finances

**Target Users:** Groups of friends, roommates, travel parties, or any collaborative group needing to manage shared expenses.

---

## 🛠 Technology Stack

### Frontend
- **HTML5** - Semantic markup
- **CSS3** - Responsive styling
- **JavaScript (Vanilla)** - Client-side logic
- **Local Storage** - Client-side data persistence

### Backend
- **Go 1.22** - High-performance backend runtime
- **Gin Framework** - Lightweight web framework with routing
- **PostgreSQL** - Relational database
- **JWT (JSON Web Tokens)** - Authentication & authorization
- **CORS** - Cross-Origin Resource Sharing support

### Database
- **PostgreSQL** - Primary data store
- **SQL** - Database schema and queries

---

## 📁 Project Structure

```
Bill-Splitting/
├── frontend/                          # Frontend web application
│   ├── dashboard.html                 # Main dashboard page
│   ├── index.html                     # Landing/home page
│   ├── login.html                     # User login page
│   ├── register.html                  # User registration page
│   ├── styles.css                     # Global styles
│   └── scripts/                       # JavaScript logic
│       ├── dashboard.js               # Dashboard functionality
│       ├── login.js                   # Login logic
│       └── register.js                # Registration logic
│
├── go-backend/                        # Go backend server
│   ├── main.go                        # Application entry point
│   ├── go.mod                         # Go module dependencies
│   ├── init.sql                       # Database initialization script
│   ├── config/                        # Configuration modules
│   │   ├── db.go                      # Database connection setup
│   │   ├── env.go                     # Environment variable handling
│   │   └── init.go                    # Initialization logic
│   └── internals/                     # Internal packages (not exported)
│       ├── handlers/                  # HTTP request handlers
│       │   ├── authz.go               # Authentication & authorization
│       │   ├── expense.go             # Expense management endpoints
│       │   ├── group.go               # Group management endpoints
│       │   └── user.go                # User management endpoints
│       ├── middlewares/               # HTTP middleware
│       │   └── auth_middle.go         # Authentication middleware
│       ├── models/                    # Data models
│       │   ├── balance.go             # Balance calculations
│       │   ├── expense.go             # Expense model
│       │   ├── group.go               # Group model
│       │   ├── member.go              # Group member model
│       │   ├── settlement.go          # Settlement tracking model
│       │   ├── split.go               # Expense split model
│       │   └── user.go                # User model
│       └── router/                    # Routing configuration
│           ├── router.go              # Router initialization
│           └── routes.go              # Route definitions
│
└── README.md                          
```
---

## 🏗 Architecture & Pipeline

### Application Flow

```
┌─────────────────┐
│   Frontend      │
│  (HTML/CSS/JS)  │
└────────┬────────┘
         │
         │ HTTP Requests/JSON
         │
         ▼
┌─────────────────────────────────┐
│   Gin Web Framework             │
│  ├─ Router                      │
│  ├─ CORS Middleware             │
│  └─ Auth Middleware (JWT)       │
└────────┬────────────────────────┘
         │
         ▼
┌─────────────────────────────────┐
│   API Handlers                  │
│  ├─ authz.go (Auth)            │
│  ├─ user.go (User Mgmt)        │
│  ├─ group.go (Group Mgmt)      │
│  ├─ expense.go (Expenses)      │
│  └─ models/* (Business Logic)  │
└────────┬────────────────────────┘
         │
         │ SQL Queries
         │
         ▼
┌─────────────────────────────────┐
│   PostgreSQL Database           │
│  ├─ users                       │
│  ├─ groups                      │
│  ├─ members                     │
│  ├─ expenses                    │
│  ├─ expense_splits              │
│  └─ settlements                 │
└─────────────────────────────────┘
```

### Request Pipeline

1. **Client Request** → Frontend sends HTTP request with JWT token (if authenticated)
2. **CORS Handling** → Gin CORS middleware validates cross-origin requests
3. **Authentication** → Auth middleware verifies JWT token and extracts user info
4. **Routing** → Router matches URL to appropriate handler
5. **Business Logic** → Handler processes request using models
6. **Database Operation** → Query executes against PostgreSQL
7. **Response** → JSON response returned to frontend
8. **Client Update** → Frontend updates UI based on response

---

## 🗄 Database Schema

### Users Table
```sql
users
├── id (PRIMARY KEY)
├── username (TEXT, NOT NULL)
├── email (TEXT, NOT NULL, UNIQUE)
├── password (TEXT, NOT NULL)
└── phone (TEXT, NOT NULL, UNIQUE)
```

### Groups Table
```sql
groups
├── id (PRIMARY KEY)
├── name (TEXT, NOT NULL)
├── created_by (INT, FOREIGN KEY → users.id)
├── created_at (TIMESTAMPTZ)
└── updated_at (TIMESTAMPTZ)
```

### Members Table
```sql
members
├── id (PRIMARY KEY)
├── group_id (INT, FOREIGN KEY → groups.id)
├── name (TEXT, NOT NULL)
├── phone (TEXT, NOT NULL)
└── UNIQUE(group_id, phone)
```

### Expenses Table
```sql
expenses
├── id (PRIMARY KEY)
├── group_id (INT, FOREIGN KEY → groups.id)
├── description (TEXT, NOT NULL)
├── amount (NUMERIC(10,2), NOT NULL)
├── paid_by_id (INT, FOREIGN KEY → members.id)
└── created_at (TIMESTAMPTZ)
```

### Expense Splits Table
```sql
expense_splits
├── id (PRIMARY KEY)
├── expense_id (INT, FOREIGN KEY → expenses.id)
├── member_id (INT, FOREIGN KEY → members.id)
└── amount_owed (NUMERIC(10,2), NOT NULL)
```

### Settlements Table
```sql
settlements
├── id (PRIMARY KEY)
├── group_id (INT, FOREIGN KEY → groups.id)
├── from_member_id (INT, FOREIGN KEY → members.id)
├── to_member_id (INT, FOREIGN KEY → members.id)
├── amount (NUMERIC(10,2), NOT NULL)
├── settled_at (TIMESTAMPTZ)
└── created_at (TIMESTAMPTZ)
```

---

## 🚀 Setup & Installation

### Prerequisites
- **Go 1.22+** - [Download](https://golang.org/dl/)
- **PostgreSQL 12+** - [Download](https://www.postgresql.org/download/)
- **Node.js/npm** (optional, for dev server) - [Download](https://nodejs.org/)
- **Git** - Version control

### Backend Setup

#### 1. Clone Repository
```bash
git clone https://github.com/jagadeesh-2006/Bill-Splitting.git
cd Bill-Splitting/go-backend
```

#### 2. Install Dependencies
```bash
go mod download
```

#### 3. Configure Environment Variables
Create a `.env` file in `go-backend/`:
```env
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=bill_splitting

# Server
PORT=8080
GIN_MODE=release

# JWT
JWT_SECRET=your_secret_key_here
```

#### 4. Initialize Database
```bash
# Connect to PostgreSQL
psql -U postgres

# Create database
CREATE DATABASE bill_splitting;

# Exit psql
\q

# Run initialization script
psql -U postgres -d bill_splitting -f init.sql
```

#### 5. Run Backend Server
```bash
go run main.go
```
Server runs on `http://localhost:8080`

### Frontend Setup

#### 1. Navigate to Frontend Directory
```bash
cd ../frontend
```

#### 2. Serve Frontend
**Option A: Using Python (built-in)**
```bash
# Python 3
python -m http.server 3000

# Python 2
python -m SimpleHTTPServer 3000
```

**Option B: Using Node.js**
```bash
npx http-server -p 3000
```

**Option C: Direct File Access**
Open `index.html` directly in browser (limited functionality without server)

Frontend accessible at `http://localhost:3000`

---

## 📡 API Endpoints

### Authentication
| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|----------------|
| POST | `/api/auth/register` | Register new user | ❌ No |
| POST | `/api/auth/login` | User login | ❌ No |

### Users
| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|----------------|
| GET | `/api/users/profile` | Get current user profile | ✅ Yes |
| PUT | `/api/users/profile` | Update user profile | ✅ Yes |

### Groups
| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|----------------|
| POST | `/api/groups` | Create new group | ✅ Yes |
| GET | `/api/groups` | List user's groups | ✅ Yes |
| GET | `/api/groups/:id` | Get group details | ✅ Yes |
| PUT | `/api/groups/:id` | Update group | ✅ Yes |
| DELETE | `/api/groups/:id` | Delete group | ✅ Yes |

### Members
| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|----------------|
| POST | `/api/groups/:id/members` | Add member to group | ✅ Yes |
| GET | `/api/groups/:id/members` | List group members | ✅ Yes |
| DELETE | `/api/groups/:id/members/:mid` | Remove member | ✅ Yes |

### Expenses
| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|----------------|
| POST | `/api/groups/:id/expenses` | Add expense | ✅ Yes |
| GET | `/api/groups/:id/expenses` | List expenses | ✅ Yes |
| GET | `/api/groups/:id/expenses/:eid` | Get expense details | ✅ Yes |
| PUT | `/api/groups/:id/expenses/:eid` | Update expense | ✅ Yes |
| DELETE | `/api/groups/:id/expenses/:eid` | Delete expense | ✅ Yes |

### Settlements
| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|----------------|
| GET | `/api/groups/:id/balances` | Get group balances | ✅ Yes |
| GET | `/api/groups/:id/settlements` | Get settlements | ✅ Yes |
| POST | `/api/groups/:id/settlements` | Record settlement | ✅ Yes |

---

## 💻 Development Workflow

### Project Structure Overview
```
Development Cycle:
┌──────────────┐
│ Code Changes │
└──────┬───────┘
       │
       ▼
┌──────────────────┐
│ Run Tests        │
│ (when added)     │
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│ Restart Server   │
│ go run main.go   │
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│ Test Endpoints   │
│ (Postman/curl)   │
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│ Update Frontend  │
│ (if needed)      │
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│ Commit Changes   │
│ (git commit)     │
└──────────────────┘
```

### Common Development Tasks

**Add New API Endpoint:**
1. Create handler method in `internals/handlers/*.go`
2. Add route in `internals/router/routes.go`
3. Test with Postman/curl
4. Update frontend if needed

**Add New Database Feature:**
1. Create migration SQL script
2. Update models in `internals/models/`
3. Update handlers to use new model
4. Test endpoints

**Fix Authentication Issues:**
1. Check `internals/middlewares/auth_middle.go`
2. Verify JWT secret in `.env`
3. Ensure token is being sent in request headers
4. Check token expiration

**Database Debugging:**
```bash
# Connect to database
psql -U postgres -d bill_splitting

# List tables
\dt

# View table structure
\d table_name

# Run query
SELECT * FROM table_name;
```

---

## 🔐 Security Considerations

- ✅ **JWT Authentication** - Secure token-based auth
- ✅ **Password Hashing** - Passwords hashed with golang.org/x/crypto
- ✅ **CORS Protection** - Configured CORS middleware
- ✅ **SQL Injection Prevention** - Parameterized queries via ORM
- ✅ **Environment Variables** - Sensitive data in .env (not committed)

**Best Practices:**
- Never commit `.env` file
- Keep JWT_SECRET strong and unique
- Use HTTPS in production
- Validate all user inputs
- Implement rate limiting for production

---

## 📝 Contributing

1. **Fork** the repository
2. **Create** feature branch (`git checkout -b feature/amazing-feature`)
3. **Commit** changes (`git commit -m 'Add amazing feature'`)
4. **Push** to branch (`git push origin feature/amazing-feature`)
5. **Open** Pull Request

---

## 📞 Support & Documentation

For issues, questions, or suggestions:
- Open a GitHub Issue
- Check existing documentation
- Review code comments for implementation details

---

## 📄 License

This project is licensed under the MIT License - see LICENSE file for details.

---

## 👨‍💻 Author

**Jagadeesh** - [GitHub Profile](https://github.com/jagadeesh-2006)

---

## 🎉 Acknowledgments

- Gin Web Framework
- PostgreSQL
- JWT-Go
- All contributors and users

---

**Happy Expense Splitting! 💸**
