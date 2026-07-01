# CBQA CRM Application

A comprehensive Customer Relationship Management (CRM) system built with VueJs, designed to manage leads, proposals, projects, invoices, and client relationships for CBQA Global.

# CRM Backend Service (Golang)
Backend service for the CRM Application, built with Go (Golang) to provide secure, scalable, and high-performance APIs consumed by the VueJS-based frontend.

### Core Enginee
- **Go 1.22+**
- **Gin / Echo** – HTTP web framework
- **GORM** – ORM for database access
- **PostgreSQL** – Primary database
- **JWT** – Authentication & authorization
- **Validator** – Request validation

### Supporting Libraries
- **godotenv** – Environment variable management
- **zap / logrus** – Structured logging
- **bcrypt** – Password hashing
- **uuid** – Unique identifier generation
- **CORS middleware** – Cross-origin support
- **Swagger (swaggo)** – API documentation

# Getting Started
## Prerequisites
- **Go 1.22 or higher**
- **PostgreSQL / MySQL**
- **Git**

# Installation
## 1. Clone the repository:
git clone <repository-url>
cd crm-backend

## 2. Install dependencies:
go mod tidy

## 3. Configure environment variables: 
Copy example env file:
cp .env.example .env

## 4. Run database migrations:
go run cmd/server/main.go migrate

## 5. Start the server:
go run cmd/server/main.go

Server will run at:
http://localhost:4000

## Features

### Core Modules

- **Dashboard** - Overview of key metrics and activities
- **Leads Management** - Track and manage leads with Kanban board view
- **Prospects & Clients** - Manage prospect and client information
- **Proposals** - Create, edit, and manage business proposals with workflow approval
- **Projects** - Track project details, progress, and deliverables
- **Invoices** - Generate and manage invoices
- **Payments** - Track payment records and status
- **Tasks** - Manage tasks and assignments
- **Expenses** - Track and categorize business expenses

### Administrative Features

- **Role Management** - Create and manage user roles
- **Permission Management** - Configure module and capability permissions
- **Approval Workflow** - Set up approval workflows for proposals and documents
- **Email Templates** - Create and manage email templates
- **Email Configuration** - Configure email settings and view email logs
- **Rate Management** - Manage service rates and pricing
- **Master Currency** - Configure currency settings
- **Format Number** - Customize number formatting
- **Expense Categories** - Manage expense categorization
- **Proposal Stages** - Configure proposal stage workflows
- **Type Unit Management** - Manage unit types
- **Template Management** - Create and manage document templates

### Reporting

- Prospects Reports
- Leads Reports
- Proposals Reports
- Projects Reports

### Authentication & Security

- User authentication with JWT tokens
- Role-based access control (RBAC)
- Protected routes
- Password reset functionality
- Session management

