#!/bin/sh

set -e

# Run the make web command.
make web

# Delete the public directory if found.
rm -rf public || true

# Clone original repository.
git clone git@github.com:kanopi/templr.git public

# Attempt to switch to the deployment branch.
cd public/

# Who am I?
git config user.email "${GIT_USER_EMAIL:-code@kanopi.com}"
git config user.name "${GIT_USER_NAME:-Kanopi Code}"

# Switch to Empty Orphan Branch.
git switch --orphan gh-pages

# Copy Everything in Web Directory.
cp -R ../web/* ./

# Push to Github Pages
git add --all
git commit -m "Deploy Templr Web Playground"
git push -f origin gh-pages

# Clean Up
cd ..
rm -rf public