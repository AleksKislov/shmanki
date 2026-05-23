#!/usr/bin/env bash
set -euo pipefail

# ---------------------------------------------------------------------------
# export-clean.sh
# Copies the repo to a clean folder, stripping deploy infra, specs, papers,
# and any secret/env files. Safe to push to a public GitHub repo.
#
# Usage:
#   bash scripts/export-clean.sh <destination>
#
# Example:
#   bash scripts/export-clean.sh ~/shmanki-clean
# ---------------------------------------------------------------------------

DEST="${1:-}"

if [[ -z "$DEST" ]]; then
  echo "Usage: bash scripts/export-clean.sh <destination>"
  echo "  e.g. bash scripts/export-clean.sh ~/shmanki-clean"
  exit 1
fi

SRC="$(cd "$(dirname "$0")/.." && pwd)"
DEST="$(eval echo "$DEST")"   # expand ~ if needed

if [[ -e "$DEST" ]]; then
  echo "Error: destination already exists: $DEST"
  echo "Remove it first or choose a different path."
  exit 1
fi

echo "Source : $SRC"
echo "Dest   : $DEST"
echo ""

# ---------------------------------------------------------------------------
# rsync copy — exclude everything that shouldn't be public
# ---------------------------------------------------------------------------
rsync -a \
  --exclude='.git/' \
  --exclude='.ai/' \
  --exclude='.env' \
  --exclude='.env.*' \
  --exclude='backend/.env' \
  --exclude='backend/.env.*' \
  --exclude='backend/bin/' \
  --exclude='frontend/node_modules/' \
  --exclude='frontend/build/' \
  --exclude='frontend/tmp/' \
  --exclude='frontend/dist/' \
  --exclude='docs/' \
  --exclude='specs/' \
  --exclude='papers/' \
  --exclude='scripts/' \
  "$SRC/" "$DEST/"

# ---------------------------------------------------------------------------
# Write a clean .gitignore in the destination
# ---------------------------------------------------------------------------
cat > "$DEST/.gitignore" << 'EOF'
# Secrets — never commit these
.env
.env.*
backend/.env
backend/.env.*

# Build artifacts
backend/bin/
frontend/node_modules/
frontend/build/
frontend/dist/
frontend/tmp/

# OS
.DS_Store
Thumbs.db
EOF

echo "Wrote clean .gitignore"

# ---------------------------------------------------------------------------
# Verify no secrets leaked
# ---------------------------------------------------------------------------
echo ""
echo "Checking for accidental secret files..."
FOUND=0
for f in ".env" ".env.prod" "backend/.env" "backend/.env.prod"; do
  if [[ -e "$DEST/$f" ]]; then
    echo "  WARNING: found $f in destination — removing it"
    rm "$DEST/$f"
    FOUND=1
  fi
done
if [[ $FOUND -eq 0 ]]; then
  echo "  OK — no secret files found"
fi

# ---------------------------------------------------------------------------
# Done
# ---------------------------------------------------------------------------
echo ""
echo "Clean copy is ready at: $DEST"
echo ""
echo "Next steps:"
echo "  cd $DEST"
echo "  git init -b main"
echo "  git add ."
echo "  git commit -m \"initial commit\""
echo "  git remote add origin git@github.com:<YOU>/shmanki.git"
echo "  git push -u origin main"
