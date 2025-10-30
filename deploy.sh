#!/bin/bash
# Railway Deployment Script for Invoice OCR Service
# This script helps deploy the service to Railway

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Print header
echo "================================================"
echo "  Invoice OCR Service - Railway Deployment"
echo "================================================"
echo ""

# Check if Railway CLI is installed
if ! command -v railway &> /dev/null; then
    print_error "Railway CLI is not installed"
    echo ""
    echo "Install it with:"
    echo "  npm install -g @railway/cli"
    echo ""
    echo "Or visit: https://docs.railway.app/develop/cli"
    exit 1
fi

print_success "Railway CLI is installed"

# Check if logged in to Railway
if ! railway whoami &> /dev/null; then
    print_warning "Not logged in to Railway"
    print_info "Logging in to Railway..."
    railway login
else
    print_success "Already logged in to Railway"
fi

# Display current Railway status
print_info "Current Railway project:"
railway status || print_warning "No Railway project linked yet"
echo ""

# Menu
echo "What would you like to do?"
echo ""
echo "1) Initialize new Railway project"
echo "2) Link to existing Railway project"
echo "3) Deploy to Railway"
echo "4) Set environment variables"
echo "5) View deployment logs"
echo "6) Open Railway dashboard"
echo "7) Test local build"
echo "8) Exit"
echo ""
read -p "Select an option (1-8): " option

case $option in
    1)
        print_info "Initializing new Railway project..."
        railway init
        print_success "Project initialized!"
        print_info "Next steps:"
        echo "  1. Set your GEMINI_API_KEY: railway variables set GEMINI_API_KEY=your-key"
        echo "  2. Deploy: railway up"
        ;;

    2)
        print_info "Linking to existing Railway project..."
        railway link
        print_success "Project linked!"
        ;;

    3)
        print_info "Deploying to Railway..."
        echo ""

        # Check if railway.toml exists
        if [ ! -f "railway.toml" ]; then
            print_error "railway.toml not found!"
            exit 1
        fi

        # Check if Dockerfile exists
        if [ ! -f "Dockerfile" ]; then
            print_error "Dockerfile not found!"
            exit 1
        fi

        print_info "Configuration files found"
        print_info "Starting deployment..."
        echo ""

        railway up

        print_success "Deployment initiated!"
        print_info "View logs with: railway logs"
        print_info "Open dashboard: railway open"
        ;;

    4)
        print_info "Setting environment variables..."
        echo ""
        echo "Required variables:"
        echo "  - GEMINI_API_KEY (required for AI processing)"
        echo ""
        echo "Optional variables:"
        echo "  - AI_DEFAULT_PROVIDER (default: gemini)"
        echo "  - TESSERACT_LANG (default: spa+eng)"
        echo "  - MAX_FILE_SIZE (default: 10485760)"
        echo "  - PROCESSING_TIMEOUT (default: 30)"
        echo ""

        read -p "Enter GEMINI_API_KEY: " gemini_key

        if [ -n "$gemini_key" ]; then
            railway variables set GEMINI_API_KEY="$gemini_key"
            print_success "GEMINI_API_KEY set successfully"
        else
            print_warning "No API key provided"
        fi

        read -p "Set additional variables? (y/N): " set_more
        if [[ $set_more =~ ^[Yy]$ ]]; then
            read -p "AI_DEFAULT_PROVIDER (gemini/openai/ollama): " provider
            if [ -n "$provider" ]; then
                railway variables set AI_DEFAULT_PROVIDER="$provider"
            fi

            read -p "TESSERACT_LANG (e.g., spa+eng): " lang
            if [ -n "$lang" ]; then
                railway variables set TESSERACT_LANG="$lang"
            fi
        fi

        print_success "Environment variables configured"
        ;;

    5)
        print_info "Fetching deployment logs..."
        railway logs
        ;;

    6)
        print_info "Opening Railway dashboard..."
        railway open
        ;;

    7)
        print_info "Testing local Docker build..."
        echo ""

        # Check if Docker is running
        if ! docker info &> /dev/null; then
            print_error "Docker is not running"
            exit 1
        fi

        print_info "Building Docker image..."
        docker build -t invoice-ocr-service:test .

        if [ $? -eq 0 ]; then
            print_success "Docker build successful!"
            echo ""
            print_info "To run locally:"
            echo "  docker run -p 8080:8080 -e GEMINI_API_KEY=your-key invoice-ocr-service:test"
        else
            print_error "Docker build failed"
            exit 1
        fi
        ;;

    8)
        print_info "Exiting..."
        exit 0
        ;;

    *)
        print_error "Invalid option"
        exit 1
        ;;
esac

echo ""
print_success "Done!"
