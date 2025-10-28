#!/bin/bash

# Test script for TERA network with 3 nodes

echo "=== TERA Network Test ==="
echo
echo "This script will start 3 nodes:"
echo "  Node 1 (bootstrap): Interested in 'machine learning'"
echo "  Node 2: Interested in 'machine learning'"
echo "  Node 3: Interested in 'cooking recipes'"
echo
echo "We'll publish ML content and see that only nodes 1 & 2 receive it."
echo
echo "Press Ctrl+C in each terminal when done"
echo
echo "Starting in 3 seconds..."
sleep 3

# Build if needed
if [ ! -f "./tera-node" ]; then
    echo "Building tera-node..."
    go build -o tera-node ./cmd/tera-node
fi

echo
echo "=== Instructions ==="
echo
echo "1. Start Node 1 (bootstrap) in terminal 1:"
echo "   ./tera-node -port 9000 -interests 'machine learning'"
echo
echo "2. Wait for Node 1 to print its peer ID, then start Node 2 in terminal 2:"
echo "   ./tera-node -port 9001 -bootstrap '/ip4/127.0.0.1/tcp/9000/p2p/<NODE1-PEER-ID>' -interests 'machine learning'"
echo
echo "3. Start Node 3 in terminal 3 (same bootstrap):"
echo "   ./tera-node -port 9002 -bootstrap '/ip4/127.0.0.1/tcp/9000/p2p/<NODE1-PEER-ID>' -interests 'cooking recipes'"
echo
echo "4. In Node 1, publish ML content:"
echo "   > publish neural networks and deep learning"
echo
echo "5. Check Node 2 - it should receive the content (ML interest matches)"
echo "   Check Node 3 - it should NOT receive it (cooking interest doesn't match)"
echo
echo "6. Try publishing cooking content from Node 3:"
echo "   > publish italian pasta recipes"
echo
echo "7. Check that only Node 3 stores it (others filter it out)"
echo
