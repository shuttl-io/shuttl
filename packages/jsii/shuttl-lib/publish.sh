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
DIST_DIR="${SCRIPT_DIR}/dist-jsii"

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
            echo "  MAVEN_USERNAME         Sonatype Central Portal username"
            echo "  MAVEN_PASSWORD         Sonatype Central Portal token/password"
            echo "  MAVEN_GPG_PRIVATE_KEY  GPG private key for signing (ASCII-armored)"
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
    
    # Fix hard links in the tarball by extracting, copying with dereference, and repacking
    log_info "Fixing hard links in tarball..."
    local tarball_name=$(basename "${tarball}")
    local temp_dir=$(mktemp -d)
    local extract_dir="${temp_dir}/extract"
    local copy_dir="${temp_dir}/copy"
    
    mkdir -p "${extract_dir}" "${copy_dir}"
    
    # Extract the tarball
    tar -xzf "${tarball}" -C "${extract_dir}"
    
    # Copy with dereference to replace hard links with actual files
    cp -rL "${extract_dir}"/* "${copy_dir}/"
    
    # Recreate the tarball without hard links and without ./ prefix
    (cd "${copy_dir}" && tar -czf "${tarball}" *)
    
    # Cleanup temp directory
    rm -rf "${temp_dir}"
    
    log_info "Hard links fixed in tarball"
    
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
# Java/Maven Central Publishing (Sonatype Central Portal)
# ==============================================================================

# Generate checksums for a file (MD5, SHA1, SHA256, SHA512)
generate_checksums() {
    local file="$1"
    
    if command -v md5sum &> /dev/null; then
        md5sum "${file}" | awk '{print $1}' > "${file}.md5"
    elif command -v md5 &> /dev/null; then
        md5 -q "${file}" > "${file}.md5"
    fi
    
    if command -v sha1sum &> /dev/null; then
        sha1sum "${file}" | awk '{print $1}' > "${file}.sha1"
    elif command -v shasum &> /dev/null; then
        shasum -a 1 "${file}" | awk '{print $1}' > "${file}.sha1"
    fi
    
    if command -v sha256sum &> /dev/null; then
        sha256sum "${file}" | awk '{print $1}' > "${file}.sha256"
    elif command -v shasum &> /dev/null; then
        shasum -a 256 "${file}" | awk '{print $1}' > "${file}.sha256"
    fi
    
    if command -v sha512sum &> /dev/null; then
        sha512sum "${file}" | awk '{print $1}' > "${file}.sha512"
    elif command -v shasum &> /dev/null; then
        shasum -a 512 "${file}" | awk '{print $1}' > "${file}.sha512"
    fi
}

# Sign a file with GPG
sign_file() {
    local file="$1"
    local passphrase="${MAVEN_GPG_PASSPHRASE:-}"
    
    if [[ -n "${passphrase}" ]]; then
        echo "${passphrase}" | gpg --batch --yes --pinentry-mode loopback \
            --passphrase-fd 0 --armor --detach-sign "${file}"
    else
        gpg --batch --yes --armor --detach-sign "${file}"
    fi
}

# Import GPG private key if provided via environment variable
import_gpg_key() {
    local private_key="${MAVEN_GPG_PRIVATE_KEY:-}"
    
    if [[ -z "${private_key}" ]]; then
        # No key to import, assume key is already in keyring
        return 0
    fi
    
    log_info "Importing GPG private key from MAVEN_GPG_PRIVATE_KEY..."
    
    local passphrase="${MAVEN_GPG_PASSPHRASE:-}"
    
    # Create a temporary file for the key
    local key_file=$(mktemp)
    echo "${private_key}" > "${key_file}"
    
    # Import the key
    if [[ -n "${passphrase}" ]]; then
        echo "${passphrase}" | gpg --batch --yes --pinentry-mode loopback \
            --passphrase-fd 0 --import "${key_file}" 2>/dev/null
    else
        gpg --batch --yes --import "${key_file}" 2>/dev/null
    fi
    
    local import_result=$?
    
    # Clean up the temporary file
    rm -f "${key_file}"
    
    if [[ ${import_result} -eq 0 ]]; then
        log_info "GPG key imported successfully"
        
        # Trust the imported key (get the key ID and set trust)
        local key_id=$(gpg --list-secret-keys --keyid-format LONG 2>/dev/null | grep -oP '(?<=sec\s{3}rsa\d{4}/)[A-F0-9]+' | head -1)
        if [[ -n "${key_id}" ]]; then
            echo -e "5\ny\n" | gpg --batch --command-fd 0 --expert --edit-key "${key_id}" trust 2>/dev/null || true
        fi
    else
        log_warning "GPG key import may have failed (key might already exist)"
    fi
    
    return 0
}

publish_java() {
    log_info "Publishing Java package to Maven Central (Sonatype Central Portal)..."
    
    local java_dir="${DIST_DIR}/java"
    if [[ ! -d "${java_dir}" ]]; then
        log_warning "No Java package found in ${java_dir}"
        return 1
    fi
    
    # Check for required tools
    if ! command -v gpg &> /dev/null; then
        log_error "GPG is not installed. GPG signing is required for Maven Central."
        return 1
    fi
    
    if ! command -v curl &> /dev/null; then
        log_error "curl is not installed. Required for Central Portal API."
        return 1
    fi
    
    # Import GPG key if provided via environment variable
    import_gpg_key
    
    # Check for credentials
    local username="${MAVEN_USERNAME:-}"
    local password="${MAVEN_PASSWORD:-}"
    
    if [[ -z "${username}" ]] || [[ -z "${password}" ]]; then
        log_error "Sonatype credentials not set. Set MAVEN_USERNAME and MAVEN_PASSWORD"
        return 1
    fi
    
    # Find all version directories containing artifacts
    local pom_files=$(find "${java_dir}" -name "*.pom")
    if [[ -z "${pom_files}" ]]; then
        log_warning "No POM files found in ${java_dir}"
        return 1
    fi
    
    # Create a temporary directory for the bundle
    local bundle_dir=$(mktemp -d)
    local bundle_zip="${bundle_dir}/bundle.zip"
    
    log_info "Creating deployment bundle..."
    
    # Process each POM file (there may be multiple for multi-module projects)
    while IFS= read -r pom_file; do
        local artifact_dir=$(dirname "${pom_file}")
        local pom_name=$(basename "${pom_file}")
        local base_name="${pom_name%.pom}"
        
        log_info "Processing: ${pom_file}"
        
        # Extract groupId, artifactId, version from POM for directory structure
        local group_id=$(grep -oPm1 "(?<=<groupId>)[^<]+" "${pom_file}" | head -1)
        local artifact_id=$(grep -oPm1 "(?<=<artifactId>)[^<]+" "${pom_file}" | head -1)
        local version=$(grep -oPm1 "(?<=<version>)[^<]+" "${pom_file}" | head -1)
        
        if [[ -z "${group_id}" ]] || [[ -z "${artifact_id}" ]] || [[ -z "${version}" ]]; then
            log_warning "Could not extract Maven coordinates from ${pom_file}"
            continue
        fi
        
        log_info "  Group ID: ${group_id}"
        log_info "  Artifact ID: ${artifact_id}"
        log_info "  Version: ${version}"
        
        # Create the Maven repository structure: groupId/artifactId/version/
        local group_path="${group_id//./\/}"
        local target_dir="${bundle_dir}/${group_path}/${artifact_id}/${version}"
        mkdir -p "${target_dir}"
        
        # Find all artifacts for this version
        local main_jar=$(find "${artifact_dir}" -maxdepth 1 -name "${base_name}.jar" 2>/dev/null | head -1)
        local sources_jar=$(find "${artifact_dir}" -maxdepth 1 -name "*-sources.jar" 2>/dev/null | head -1)
        local javadoc_jar=$(find "${artifact_dir}" -maxdepth 1 -name "*-javadoc.jar" 2>/dev/null | head -1)
        
        log_info "  Main JAR: ${main_jar:-not found}"
        log_info "  Sources JAR: ${sources_jar:-not found}"
        log_info "  Javadoc JAR: ${javadoc_jar:-not found}"
        
        # Copy and process POM
        local target_pom="${target_dir}/${artifact_id}-${version}.pom"
        cp "${pom_file}" "${target_pom}"
        log_info "  Signing and checksumming POM..."
        sign_file "${target_pom}" || { log_error "Failed to sign POM"; rm -rf "${bundle_dir}"; return 1; }
        generate_checksums "${target_pom}"
        
        # Copy and process main JAR
        if [[ -n "${main_jar}" ]] && [[ -f "${main_jar}" ]]; then
            local target_jar="${target_dir}/${artifact_id}-${version}.jar"
            cp "${main_jar}" "${target_jar}"
            log_info "  Signing and checksumming main JAR..."
            sign_file "${target_jar}" || { log_error "Failed to sign JAR"; rm -rf "${bundle_dir}"; return 1; }
            generate_checksums "${target_jar}"
        fi
        
        # Copy and process sources JAR
        if [[ -n "${sources_jar}" ]] && [[ -f "${sources_jar}" ]]; then
            local target_sources="${target_dir}/${artifact_id}-${version}-sources.jar"
            cp "${sources_jar}" "${target_sources}"
            log_info "  Signing and checksumming sources JAR..."
            sign_file "${target_sources}" || { log_error "Failed to sign sources JAR"; rm -rf "${bundle_dir}"; return 1; }
            generate_checksums "${target_sources}"
        fi
        
        # Copy and process javadoc JAR
        if [[ -n "${javadoc_jar}" ]] && [[ -f "${javadoc_jar}" ]]; then
            local target_javadoc="${target_dir}/${artifact_id}-${version}-javadoc.jar"
            cp "${javadoc_jar}" "${target_javadoc}"
            log_info "  Signing and checksumming javadoc JAR..."
            sign_file "${target_javadoc}" || { log_error "Failed to sign javadoc JAR"; rm -rf "${bundle_dir}"; return 1; }
            generate_checksums "${target_javadoc}"
        fi
        
    done <<< "${pom_files}"
    
    # Create the bundle ZIP (excluding the bundle.zip itself)
    log_info "Creating bundle ZIP..."
    (cd "${bundle_dir}" && zip -r "${bundle_zip}" . -x "bundle.zip")
    
    if [[ "${DRY_RUN}" == "true" ]]; then
        log_info "[DRY RUN] Would upload bundle to Sonatype Central Portal"
        log_info "[DRY RUN] Bundle contents:"
        unzip -l "${bundle_zip}" | head -50
        rm -rf "${bundle_dir}"
        return 0
    fi
    
    # Upload to Sonatype Central Portal
    log_info "Uploading bundle to Sonatype Central Portal..."
    
    # Create the authorization header (Basic auth with username:password)
    local auth_token=$(echo -n "${username}:${password}" | base64)
    
    # Upload the bundle
    local upload_response
    upload_response=$(curl -s -w "\n%{http_code}" \
        -X POST "https://central.sonatype.com/api/v1/publisher/upload" \
        -H "Authorization: Bearer ${auth_token}" \
        -F "bundle=@${bundle_zip}")
    
    local http_code=$(echo "${upload_response}" | tail -1)
    local response_body=$(echo "${upload_response}" | sed '$d')
    
    if [[ "${http_code}" != "201" ]] && [[ "${http_code}" != "200" ]]; then
        log_error "Failed to upload bundle. HTTP ${http_code}"
        log_error "Response: ${response_body}"
        rm -rf "${bundle_dir}"
        return 1
    fi
    
    # Extract deployment ID from response
    local deployment_id="${response_body}"
    log_info "Bundle uploaded successfully. Deployment ID: ${deployment_id}"
    
    # Check deployment status
    log_info "Checking deployment status..."
    local max_attempts=60
    local attempt=0
    local status="PENDING"
    
    while [[ "${status}" == "PENDING" ]] || [[ "${status}" == "VALIDATING" ]] || [[ "${status}" == "PUBLISHING" ]]; do
        sleep 5
        attempt=$((attempt + 1))
        
        if [[ ${attempt} -ge ${max_attempts} ]]; then
            log_warning "Timeout waiting for deployment validation. Check status manually at https://central.sonatype.com"
            break
        fi
        
        local status_response
        status_response=$(curl -s \
            -X POST "https://central.sonatype.com/api/v1/publisher/status?id=${deployment_id}" \
            -H "Authorization: Bearer ${auth_token}")
        
        status=$(echo "${status_response}" | grep -oP '"deploymentState"\s*:\s*"\K[^"]+' || echo "UNKNOWN")
        log_info "  Status: ${status} (attempt ${attempt}/${max_attempts})"
    done
    
    # Handle final status
    case "${status}" in
        "VALIDATED")
            log_info "Deployment validated. Publishing..."
            # Publish the deployment
            local publish_response
            publish_response=$(curl -s -w "\n%{http_code}" \
                -X POST "https://central.sonatype.com/api/v1/publisher/deployment/${deployment_id}" \
                -H "Authorization: Bearer ${auth_token}")
            
            local publish_http_code=$(echo "${publish_response}" | tail -1)
            if [[ "${publish_http_code}" == "204" ]] || [[ "${publish_http_code}" == "200" ]]; then
                log_success "Deployment published successfully!"
            else
                log_warning "Publish request returned HTTP ${publish_http_code}. Check status at https://central.sonatype.com"
            fi
            ;;
        "PUBLISHED")
            log_success "Deployment already published!"
            ;;
        "FAILED")
            log_error "Deployment validation failed. Check https://central.sonatype.com for details."
            rm -rf "${bundle_dir}"
            return 1
            ;;
        *)
            log_warning "Deployment status: ${status}. Check https://central.sonatype.com for details."
            ;;
    esac
    
    # Cleanup
    rm -rf "${bundle_dir}"
    
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
    local version=$(node -p "require('${SCRIPT_DIR}/package.json').version")
    log_info "Package version: ${version}"
    
    if [[ "${DRY_RUN}" == "true" ]]; then
        log_info "[DRY RUN] Would push Go module from ${module_dir} with tag v${version}"
    else
        # Go modules are typically published by pushing to a Git repository
        # The module path in go.mod should match the repository URL
        local go_repo="github.com/shuttl-io/shuttl-core-go"
        local temp_dir=$(mktemp -d)
        
        log_info "Cloning Go module repository..."
        local repo_cloned=false
        if git clone "https://${go_repo}.git" "${temp_dir}/repo" 2>/dev/null; then
            repo_cloned=true
        else
            # If repo doesn't exist, initialize a new one
            mkdir -p "${temp_dir}/repo"
            cd "${temp_dir}/repo"
            git init
            git remote add origin "https://${go_repo}.git"
        fi
        
        cd "${temp_dir}/repo"
        
        # Determine the default branch and ensure we're on it
        local default_branch="main"
        if [[ "${repo_cloned}" == "true" ]]; then
            # Try to determine the default branch from remote
            if git show-ref --verify --quiet refs/remotes/origin/main; then
                default_branch="main"
                git checkout -b main origin/main 2>/dev/null || git checkout main 2>/dev/null || true
            elif git show-ref --verify --quiet refs/remotes/origin/master; then
                default_branch="master"
                git checkout -b master origin/master 2>/dev/null || git checkout master 2>/dev/null || true
            else
                # No remote branches, create main
                git checkout -b main 2>/dev/null || true
            fi
        else
            # New repo, create main branch
            git checkout -b main 2>/dev/null || true
        fi
        
        # Copy generated Go files
        cp -r "${module_dir}"/* "${temp_dir}/repo/"
        
        # Add all files
        git add -A
        
        # Check if there are any changes to commit
        local has_changes=false
        if ! git diff --staged --quiet || ! git diff --quiet; then
            has_changes=true
        fi
        
        if [[ "${has_changes}" == "true" ]]; then
            git commit -m "Release v${version}"
            log_info "Committed changes for release v${version}"
        else
            log_info "No file changes detected, but will still tag and push version v${version}"
        fi
        
        # Always create/update the tag (force update if it exists)
        # Tag should point to current HEAD (whether it's a new commit or existing)
        git tag -a "v${version}" -m "Release v${version}" -f
        log_info "Created/updated tag v${version}"
        
        # Push with authentication if token is available
        if [[ -n "${GO_REPO_TOKEN:-}" ]]; then
            git remote set-url origin "https://x-access-token:${GO_REPO_TOKEN}@${go_repo}.git"
        fi
        
        # Always push commits (will be no-op if nothing to push)
        log_info "Pushing commits to ${default_branch}..."
        git push origin "${default_branch}" || log_warning "No commits to push or push failed"
        
        # Always push the tag (force push to update if it exists)
        log_info "Pushing tag v${version}..."
        git push origin "v${version}" --force
        
        # Cleanup
        rm -rf "${temp_dir}"
    fi
    
    log_success "Go module published successfully"
}

# ==============================================================================
# GitHub Release Artifact Upload
# ==============================================================================
upload_artifacts_to_github_release() {
    # Check if we're running in GitHub Actions
    if [[ -z "${GITHUB_ACTIONS:-}" ]]; then
        log_info "Not running in GitHub Actions, skipping artifact upload"
        return 0
    fi
    
    # Check if gh CLI is available
    if ! command -v gh &> /dev/null; then
        log_warning "GitHub CLI (gh) is not available, skipping artifact upload"
        return 0
    fi
    
    # Get version from git tag - only upload if there's a release tag
    local version
    local tag
    tag=$(git tag --points-at HEAD 2>/dev/null | head -1)
    if [[ -z "${tag}" ]]; then
        log_info "No release tag found at HEAD, skipping artifact upload"
        return 0
    fi
    version="${tag}"
    # Extract version number (remove 'v' prefix if present)
    local version_num="${version#v}"
    
    log_info "Uploading artifacts to GitHub release: ${version}"
    
    # Language directories to check
    local lang_dirs=("js" "dotnet" "python" "java" "go")
    local uploaded=0
    local temp_dir=$(mktemp -d)
    
    # Process each language directory
    for lang_dir in "${lang_dirs[@]}"; do
        local lang_path="${DIST_DIR}/${lang_dir}"
        
        # Skip if directory doesn't exist
        if [[ ! -d "${lang_path}" ]]; then
            continue
        fi
        
        # Count files in the directory
        local file_count=$(find "${lang_path}" -type f | wc -l)
        
        if [[ ${file_count} -eq 0 ]]; then
            log_info "No files found in ${lang_dir} directory, skipping"
            continue
        fi
        
        local artifact_file
        
        if [[ ${file_count} -gt 1 ]]; then
            # Multiple files: create a tarball
            local tarball_name="${lang_dir}-core@${version_num}.tar.gz"
            local tarball_path="${temp_dir}/${tarball_name}"
            
            log_info "Creating tarball for ${lang_dir} (${file_count} files): ${tarball_name}"
            (cd "${DIST_DIR}" && tar -czf "${tarball_path}" "${lang_dir}")
            artifact_file="${tarball_path}"
        else
            # Single file: use it directly
            local single_file=$(find "${lang_path}" -type f | head -1)
            artifact_file="${single_file}"
            log_info "Found single file in ${lang_dir}: $(basename "${artifact_file}")"
        fi
        
        # Upload the artifact
        if [[ -f "${artifact_file}" ]]; then
            log_info "Uploading: $(basename "${artifact_file}")"
            if gh release upload "${version}" "${artifact_file}" --clobber 2>/dev/null; then
                uploaded=$((uploaded + 1))
                log_success "Uploaded: $(basename "${artifact_file}")"
            else
                log_warning "Failed to upload: $(basename "${artifact_file}")"
            fi
        fi
    done
    
    # Cleanup temp directory
    rm -rf "${temp_dir}"
    
    if [[ ${uploaded} -gt 0 ]]; then
        log_success "Uploaded ${uploaded} artifact(s) to GitHub release ${version}"
    else
        log_warning "No artifacts were successfully uploaded"
    fi
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
    
    # Upload artifacts to GitHub release if running in GitHub Actions
    upload_artifacts_to_github_release
    
    exit 0
else
    log_error "${FAILED} package(s) failed to publish"
    exit 1
fi

