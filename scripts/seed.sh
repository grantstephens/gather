#!/bin/bash
# Seed dummy data for Gather

BASE_URL="${BASE_URL:-http://127.0.0.1:8090}"

echo "Seeding data to $BASE_URL..."

# Wait for server to be ready
for i in {1..10}; do
    if curl -s "$BASE_URL/api/health" > /dev/null 2>&1; then
        break
    fi
    echo "Waiting for server..."
    sleep 1
done

# Create test users
echo "Creating users..."
curl -s -X POST "$BASE_URL/api/collections/users/records" \
    -H "Content-Type: application/json" \
    -d '{"email":"editor@example.com","password":"editorpassword123","passwordConfirm":"editorpassword123","role":"editor","display_name":"Editor"}' > /dev/null
echo "  Editor: editor@example.com / editorpassword123"

curl -s -X POST "$BASE_URL/api/collections/users/records" \
    -H "Content-Type: application/json" \
    -d '{"email":"admin@example.com","password":"adminpassword123","passwordConfirm":"adminpassword123","role":"admin","display_name":"Admin"}' > /dev/null
echo "  Admin:  admin@example.com / adminpassword123"

curl -s -X POST "$BASE_URL/api/collections/users/records" \
    -H "Content-Type: application/json" \
    -d '{"email":"user@example.com","password":"userpassword123","passwordConfirm":"userpassword123","role":"user","display_name":"Test User"}' > /dev/null
echo "  User:  user@example.com / userpassword123"

# Create tags
echo "Creating tags..."
curl -s -X POST "$BASE_URL/api/collections/tags/records" \
    -H "Content-Type: application/json" \
    -d '{"name":"music","color":"#e74c3c"}' > /dev/null

curl -s -X POST "$BASE_URL/api/collections/tags/records" \
    -H "Content-Type: application/json" \
    -d '{"name":"art","color":"#9b59b6"}' > /dev/null

curl -s -X POST "$BASE_URL/api/collections/tags/records" \
    -H "Content-Type: application/json" \
    -d '{"name":"community","color":"#3498db"}' > /dev/null

curl -s -X POST "$BASE_URL/api/collections/tags/records" \
    -H "Content-Type: application/json" \
    -d '{"name":"workshop","color":"#2ecc71"}' > /dev/null

curl -s -X POST "$BASE_URL/api/collections/tags/records" \
    -H "Content-Type: application/json" \
    -d '{"name":"food","color":"#f39c12"}' > /dev/null

# Create places
echo "Creating places..."
PLACE1=$(curl -s -X POST "$BASE_URL/api/collections/places/records" \
    -H "Content-Type: application/json" \
    -d '{"name":"Community Center","address":"123 Main St","city":"Portland","latitude":45.5152,"longitude":-122.6784}' | grep -o '"id":"[^"]*"' | cut -d'"' -f4)

PLACE2=$(curl -s -X POST "$BASE_URL/api/collections/places/records" \
    -H "Content-Type: application/json" \
    -d '{"name":"The Art Space","address":"456 Gallery Ave","city":"Portland","latitude":45.5231,"longitude":-122.6765}' | grep -o '"id":"[^"]*"' | cut -d'"' -f4)

PLACE3=$(curl -s -X POST "$BASE_URL/api/collections/places/records" \
    -H "Content-Type: application/json" \
    -d '{"name":"Riverside Park","address":"789 River Rd","city":"Portland","latitude":45.5289,"longitude":-122.6951}' | grep -o '"id":"[^"]*"' | cut -d'"' -f4)

echo "  Places: $PLACE1, $PLACE2, $PLACE3"

# Get tag IDs
MUSIC_TAG=$(curl -s "$BASE_URL/api/collections/tags/records?filter=name='music'" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
ART_TAG=$(curl -s "$BASE_URL/api/collections/tags/records?filter=name='art'" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
COMMUNITY_TAG=$(curl -s "$BASE_URL/api/collections/tags/records?filter=name='community'" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
WORKSHOP_TAG=$(curl -s "$BASE_URL/api/collections/tags/records?filter=name='workshop'" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
FOOD_TAG=$(curl -s "$BASE_URL/api/collections/tags/records?filter=name='food'" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

echo "  Tags: music=$MUSIC_TAG, art=$ART_TAG, community=$COMMUNITY_TAG"

# Calculate dates (works on both Linux and macOS)
if date -d "+1 day" +%Y-%m-%d > /dev/null 2>&1; then
    # Linux date
    TOMORROW=$(date -d "+1 day" +%Y-%m-%d)
    NEXT_WEEK=$(date -d "+7 days" +%Y-%m-%d)
    IN_TWO_WEEKS=$(date -d "+14 days" +%Y-%m-%d)
    NEXT_MONTH=$(date -d "+30 days" +%Y-%m-%d)
else
    # macOS date
    TOMORROW=$(date -v+1d +%Y-%m-%d)
    NEXT_WEEK=$(date -v+7d +%Y-%m-%d)
    IN_TWO_WEEKS=$(date -v+14d +%Y-%m-%d)
    NEXT_MONTH=$(date -v+30d +%Y-%m-%d)
fi

# Download placeholder images
echo "Downloading event images..."
IMG_DIR=$(mktemp -d)
curl -sL "https://picsum.photos/seed/openmic/800/600" -o "$IMG_DIR/openmic.jpg"
curl -sL "https://picsum.photos/seed/drawing/800/600" -o "$IMG_DIR/drawing.jpg"
curl -sL "https://picsum.photos/seed/potluck/800/600" -o "$IMG_DIR/potluck.jpg"
curl -sL "https://picsum.photos/seed/jazz/800/600" -o "$IMG_DIR/jazz.jpg"
curl -sL "https://picsum.photos/seed/pottery/800/600" -o "$IMG_DIR/pottery.jpg"
curl -sL "https://picsum.photos/seed/cleanup/800/600" -o "$IMG_DIR/cleanup.jpg"

# Create events with images (using multipart form data)
echo "Creating events..."

curl -s -X POST "$BASE_URL/api/collections/events/records" \
    -F "title=Open Mic Night" \
    -F "description=<p>Join us for an evening of local talent! Singers, poets, comedians all welcome. Sign up at the door.</p><p>Free entry, donations appreciated.</p>" \
    -F "start_datetime=${TOMORROW}T19:00:00Z" \
    -F "end_datetime=${TOMORROW}T22:00:00Z" \
    -F "place=$PLACE1" \
    -F "tags=$MUSIC_TAG" \
    -F "tags=$COMMUNITY_TAG" \
    -F "status=published" \
    -F "image=@$IMG_DIR/openmic.jpg" > /dev/null
echo "  Created: Open Mic Night"

curl -s -X POST "$BASE_URL/api/collections/events/records" \
    -F "title=Figure Drawing Workshop" \
    -F "description=<p>Weekly figure drawing session with live models. All skill levels welcome.</p><p>Bring your own materials or use ours for a small fee.</p>" \
    -F "start_datetime=${NEXT_WEEK}T14:00:00Z" \
    -F "end_datetime=${NEXT_WEEK}T17:00:00Z" \
    -F "place=$PLACE2" \
    -F "tags=$ART_TAG" \
    -F "tags=$WORKSHOP_TAG" \
    -F "status=published" \
    -F "image=@$IMG_DIR/drawing.jpg" > /dev/null
echo "  Created: Figure Drawing Workshop"

curl -s -X POST "$BASE_URL/api/collections/events/records" \
    -F "title=Community Potluck" \
    -F "description=<p>Monthly community potluck in the park! Bring a dish to share and meet your neighbors.</p><p>Tables and chairs provided. Bring your own plates and utensils.</p>" \
    -F "start_datetime=${NEXT_WEEK}T12:00:00Z" \
    -F "end_datetime=${NEXT_WEEK}T15:00:00Z" \
    -F "place=$PLACE3" \
    -F "tags=$COMMUNITY_TAG" \
    -F "tags=$FOOD_TAG" \
    -F "status=published" \
    -F "image=@$IMG_DIR/potluck.jpg" > /dev/null
echo "  Created: Community Potluck"

curl -s -X POST "$BASE_URL/api/collections/events/records" \
    -F "title=Jazz in the Park" \
    -F "description=<p>Free jazz concert featuring local musicians. Bring a blanket and enjoy the music!</p><p>Food trucks will be on site.</p>" \
    -F "start_datetime=${IN_TWO_WEEKS}T18:00:00Z" \
    -F "end_datetime=${IN_TWO_WEEKS}T21:00:00Z" \
    -F "place=$PLACE3" \
    -F "tags=$MUSIC_TAG" \
    -F "status=published" \
    -F "image=@$IMG_DIR/jazz.jpg" > /dev/null
echo "  Created: Jazz in the Park"

curl -s -X POST "$BASE_URL/api/collections/events/records" \
    -F "title=Pottery for Beginners" \
    -F "description=<p>Learn the basics of wheel throwing in this hands-on workshop.</p><p>All materials provided. Wear clothes you don't mind getting dirty!</p>" \
    -F "start_datetime=${NEXT_MONTH}T10:00:00Z" \
    -F "end_datetime=${NEXT_MONTH}T13:00:00Z" \
    -F "place=$PLACE2" \
    -F "tags=$ART_TAG" \
    -F "tags=$WORKSHOP_TAG" \
    -F "status=published" \
    -F "image=@$IMG_DIR/pottery.jpg" > /dev/null
echo "  Created: Pottery for Beginners"

curl -s -X POST "$BASE_URL/api/collections/events/records" \
    -F "title=Neighborhood Cleanup" \
    -F "description=<p>Help keep our neighborhood beautiful! We'll provide gloves, bags, and refreshments.</p><p>Meet at the Community Center entrance.</p>" \
    -F "start_datetime=${IN_TWO_WEEKS}T09:00:00Z" \
    -F "end_datetime=${IN_TWO_WEEKS}T12:00:00Z" \
    -F "place=$PLACE1" \
    -F "tags=$COMMUNITY_TAG" \
    -F "status=published" \
    -F "image=@$IMG_DIR/cleanup.jpg" > /dev/null
echo "  Created: Neighborhood Cleanup"

# Cleanup temp images
rm -rf "$IMG_DIR"

echo ""
echo "Seed complete! Created 3 users, 5 tags, 3 places, and 6 events."
echo ""
echo "Frontend login (/login):"
echo "  admin@example.com  / adminpassword123  (admin)"
echo "  editor@example.com / editorpassword123 (editor)"
echo "  user@example.com   / userpassword123   (user)"
