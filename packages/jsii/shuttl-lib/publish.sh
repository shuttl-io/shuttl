#!/usr/bin/env bash
set -euo pipefail

# JSII Multi-Language Package Publisher
# This script publishes packages to their respective package managers:
# - JavaScript/TypeScript -> npm
# - Python -> PyPI
# - Java -> Maven Central
# - C#/.NET -> NuGet
# - Go -> GitHub (via git push with tags)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DIST_DIR="${SCRIPT_DIR}/shuttl-lib/dist-jsii"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if dist-jsii directory exists
if [[ ! -d "${DIST_DIR}" ]]; then
    log_error "dist-jsii directory not found at ${DIST_DIR}"
    log_error "Please run 'jsii-pacmak' first to generate the packages."
    exit 1
fi

# Parse command line arguments
PUBLISH_JS=${PUBLISH_JS:-true}
PUBLISH_PYTHON=${PUBLISH_PYTHON:-true}
PUBLISH_JAVA=${PUBLISH_JAVA:-true}
PUBLISH_DOTNET=${PUBLISH_DOTNET:-true}
PUBLISH_GO=${PUBLISH_GO:-true}
DRY_RUN=${DRY_RUN:-false}

while [[ $# -gt 0 ]]; do
    case $1 in
        --js-only)
            PUBLISH_PYTHON=false
            PUBLISH_JAVA=false
            PUBLISH_DOTNET=false
            PUBLISH_GO=false
            shift
            ;;
        --python-only)
            PUBLISH_JS=false
            PUBLISH_JAVA=false
            PUBLISH_DOTNET=false
            PUBLISH_GO=false
            shift
            ;;
        --java-only)
            PUBLISH_JS=false
            PUBLISH_PYTHON=false
            PUBLISH_DOTNET=false
            PUBLISH_GO=false
            shift
            ;;
        --dotnet-only)
            PUBLISH_JS=false
            PUBLISH_PYTHON=false
            PUBLISH_JAVA=false
            PUBLISH_GO=false
            shift
            ;;
        --go-only)
            PUBLISH_JS=false
            PUBLISH_PYTHON=false
            PUBLISH_JAVA=false
            PUBLISH_DOTNET=false
            shift
            ;;
        --skip-js)
            PUBLISH_JS=false
            shift
            ;;
        --skip-python)
            PUBLISH_PYTHON=false
            shift
            ;;
        --skip-java)
            PUBLISH_JAVA=false
            shift
            ;;
        --skip-dotnet)
            PUBLISH_DOTNET=false
            shift
            ;;
        --skip-go)
            PUBLISH_GO=false
            shift
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --js-only       Only publish JavaScript package"
            echo "  --python-only   Only publish Python package"
            echo "  --java-only     Only publish Java package"
            echo "  --dotnet-only   Only publish .NET package"
            echo "  --go-only       Only publish Go package"
            echo "  --skip-js       Skip JavaScript publishing"
            echo "  --skip-python   Skip Python publishing"
            echo "  --skip-java     Skip Java publishing"
            echo "  --skip-dotnet   Skip .NET publishing"
            echo "  --skip-go       Skip Go publishing"
            echo "  --dry-run       Show what would be published without publishing"
            echo "  --help, -h      Show this help message"
            echo ""
            echo "Environment variables:"
            echo "  NPM_TOKEN              npm authentication token"
            echo "  PYPI_TOKEN             PyPI authentication token"
            echo "  MAVEN_USERNAME         Maven Central username"
            echo "  MAVEN_PASSWORD         Maven Central password"
            echo "  MAVEN_GPG_PASSPHRASE   GPG passphrase for signing Maven artifacts"
            echo "  NUGET_API_KEY          NuGet API key"
            echo "  GO_REPO_TOKEN          GitHub token for Go module repository"
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

if [[ "${DRY_RUN}" == "true" ]]; then
    log_warning "DRY RUN MODE - No packages will be published"
fi

# ==============================================================================
# JavaScript/npm Publishing
# ==============================================================================
publish_js() {
    log_info "Publishing JavaScript package to npm..."
    
    local js_dir="${DIST_DIR}/js"
    if [[ ! -d "${js_dir}" ]]; then
        log_warning "No JavaScript package found in ${js_dir}"
        return 1
    fi
    
    local tarball=$(find "${js_dir}" -name "*.tgz" | head -1)
    if [[ -z "${tarball}" ]]; then
        log_warning "No .tgz file found in ${js_dir}"
        return 1
    fi
    
    log_info "Found tarball: ${tarball}"
    
    if [[ "${DRY_RUN}" == "true" ]]; then
        log_info "[DRY RUN] Would publish: ${tarball}"
    else
        npm publish "${tarball}" --access public
    fi
    
    log_success "JavaScript package published successfully"
}

# ==============================================================================
# Python/PyPI Publishing
# ==============================================================================
publish_python() {
    log_info "Publishing Python package to PyPI..."
    
    local python_dir="${DIST_DIR}/python"
    if [[ ! -d "${python_dir}" ]]; then
        log_warning "No Python package found in ${python_dir}"
        return 1
    fi
    
    local wheel=$(find "${python_dir}" -name "*.whl" | head -1)
    local sdist=$(find "${python_dir}" -name "*.tar.gz" | head -1)
    
    if [[ -z "${wheel}" ]] && [[ -z "${sdist}" ]]; then
        log_warning "No Python distribution files found in ${python_dir}"
        return 1
    fi
    
    log_info "Found wheel: ${wheel:-none}"
    log_info "Found sdist: ${sdist:-none}"
    
    # Check for twine
    if ! command -v twine &> /dev/null; then
        log_error "twine is not installed. Install it with: pip install twine"
        return 1
    fi
    
    if [[ "${DRY_RUN}" == "true" ]]; then
        log_info "[DRY RUN] Would publish Python packages from ${python_dir}"
    else
        if [[ -n "${PYPI_TOKEN:-}" ]]; then
            uvx twine upload --username __token__ --password "${PYPI_TOKEN}" "${python_dir}"/*
        else
            uvx twine upload "${python_dir}"/*
        fi
    fi
    
    log_success "Python package published successfully"
}

# ==============================================================================
# Java/Maven Central Publishing
# ==============================================================================
publish_java() {
    log_info "Publishing Java package to Maven Central..."
    
    local java_dir="${DIST_DIR}/java"
    if [[ ! -d "${java_dir}" ]]; then
        log_warning "No Java package found in ${java_dir}"
        return 1
    fi
    
    # Find the POM file to get coordinates
    local pom_file=$(find "${java_dir}" -name "*.pom" | head -1)
    if [[ -z "${pom_file}" ]]; then
        log_warning "No POM file found in ${java_dir}"
        return 1
    fi
    
    log_info "Found POM file: ${pom_file}"
    
    # Find the corresponding JAR files
    local jar_dir=$(dirname "${pom_file}")
    local main_jar=$(find "${jar_dir}" -name "*.jar" ! -name "*-sources.jar" ! -name "*-javadoc.jar" | head -1)
    local sources_jar=$(find "${jar_dir}" -name "*-sources.jar" | head -1)
    local javadoc_jar=$(find "${jar_dir}" -name "*-javadoc.jar" | head -1)
    
    log_info "Found main JAR: ${main_jar:-none}"
    log_info "Found sources JAR: ${sources_jar:-none}"
    log_info "Found javadoc JAR: ${javadoc_jar:-none}"
    
    if [[ "${DRY_RUN}" == "true" ]]; then
        log_info "[DRY RUN] Would publish Java artifacts from ${jar_dir}"
    else
        # Check if Maven is available
        if ! command -v mvn &> /dev/null; then
            log_error "Maven is not installed. Install Maven to publish Java packages."
            return 1
        fi
        
        # Deploy using Maven deploy:deploy-file
        # This requires proper Maven settings.xml with server credentials
        if [[ -n "${main_jar}" ]]; then
            mvn deploy:deploy-file \
                -Dfile="${main_jar}" \
                -DpomFile="${pom_file}" \
                -DrepositoryId=ossrh \
                -Durl=https://s01.oss.sonatype.org/service/local/staging/deploy/maven2/ \
                ${sources_jar:+-Dsources="${sources_jar}"} \
                ${javadoc_jar:+-Djavadoc="${javadoc_jar}"}
        fi
    fi
    
    log_success "Java package published successfully"
}

# ==============================================================================
# .NET/NuGet Publishing
# ==============================================================================
publish_dotnet() {
    log_info "Publishing .NET package to NuGet..."
    
    local dotnet_dir="${DIST_DIR}/dotnet"
    if [[ ! -d "${dotnet_dir}" ]]; then
        log_warning "No .NET package found in ${dotnet_dir}"
        return 1
    fi
    
    local nupkg=$(find "${dotnet_dir}" -name "*.nupkg" ! -name "*.snupkg" | head -1)
    if [[ -z "${nupkg}" ]]; then
        log_warning "No .nupkg file found in ${dotnet_dir}"
        return 1
    fi
    
    log_info "Found NuGet package: ${nupkg}"
    
    # Find symbols package if present
    local snupkg=$(find "${dotnet_dir}" -name "*.snupkg" | head -1)
    if [[ -n "${snupkg}" ]]; then
        log_info "Found symbols package: ${snupkg}"
    fi
    
    if [[ "${DRY_RUN}" == "true" ]]; then
        log_info "[DRY RUN] Would publish: ${nupkg}"
        if [[ -n "${snupkg}" ]]; then
            log_info "[DRY RUN] Would publish symbols: ${snupkg}"
        fi
    else
        if [[ -z "${NUGET_API_KEY:-}" ]]; then
            log_error "NUGET_API_KEY environment variable is not set"
            return 1
        fi
        
        dotnet nuget push "${nupkg}" \
            --api-key "${NUGET_API_KEY}" \
            --source https://api.nuget.org/v3/index.json \
            --skip-duplicate
    fi
    
    log_success ".NET package published successfully"
}

# ==============================================================================
# Go Module Publishing
# ==============================================================================
publish_go() {
    log_info "Publishing Go module..."
    
    local go_dir="${DIST_DIR}/go"
    if [[ ! -d "${go_dir}" ]]; then
        log_warning "No Go package found in ${go_dir}"
        return 1
    fi
    
    # Find the module directory (contains go.mod)
    local go_mod=$(find "${go_dir}" -name "go.mod" | head -1)
    if [[ -z "${go_mod}" ]]; then
        log_warning "No go.mod file found in ${go_dir}"
        return 1
    fi
    
    local module_dir=$(dirname "${go_mod}")
    log_info "Found Go module at: ${module_dir}"
    
    # Read version from package.json
    local version=$(node -p "require('${SCRIPT_DIR}/shuttl-lib/package.json').version")
    log_info "Package version: ${version}"
    
    if [[ "${DRY_RUN}" == "true" ]]; then
        log_info "[DRY RUN] Would push Go module from ${module_dir} with tag v${version}"
    else
        # Go modules are typically published by pushing to a Git repository
        # The module path in go.mod should match the repository URL
        local go_repo="github.com/shuttl-io/shuttl-core-go"
        local temp_dir=$(mktemp -d)
        
        log_info "Cloning Go module repository..."
        git clone "https://${go_repo}.git" "${temp_dir}/repo" 2>/dev/null || {
            # If repo doesn't exist, initialize a new one
            mkdir -p "${temp_dir}/repo"
            cd "${temp_dir}/repo"
            git init
            git remote add origin "https://${go_repo}.git"
        }
        
        # Copy generated Go files
        cp -r "${module_dir}"/* "${temp_dir}/repo/"
        
        cd "${temp_dir}/repo"
        git add -A
        git commit -m "Release v${version}" || log_info "No changes to commit"
        git tag -a "v${version}" -m "Release v${version}" 2>/dev/null || log_warning "Tag v${version} may already exist"
        
        # Push with authentication if token is available
        if [[ -n "${GO_REPO_TOKEN:-}" ]]; then
            git remote set-url origin "https://x-access-token:${GO_REPO_TOKEN}@${go_repo}.git"
        fi
        
        git push origin main --tags || git push origin master --tags
        
        # Cleanup
        rm -rf "${temp_dir}"
    fi
    
    log_success "Go module published successfully"
}

# ==============================================================================
# Main Execution
# ==============================================================================
log_info "Starting JSII package publishing..."
log_info "Distribution directory: ${DIST_DIR}"

FAILED=0

if [[ "${PUBLISH_JS}" == "true" ]]; then
    publish_js || FAILED=$((FAILED + 1))
fi

if [[ "${PUBLISH_PYTHON}" == "true" ]]; then
    publish_python || FAILED=$((FAILED + 1))
fi

if [[ "${PUBLISH_JAVA}" == "true" ]]; then
    publish_java || FAILED=$((FAILED + 1))
fi

if [[ "${PUBLISH_DOTNET}" == "true" ]]; then
    publish_dotnet || FAILED=$((FAILED + 1))
fi

if [[ "${PUBLISH_GO}" == "true" ]]; then
    publish_go || FAILED=$((FAILED + 1))
fi

echo ""
if [[ ${FAILED} -eq 0 ]]; then
    log_success "All packages published successfully!"
    exit 0
else
    log_error "${FAILED} package(s) failed to publish"
    exit 1
fi

