#!/usr/bin/env python3
"""Seed dummy data for Gather."""

import os
import sys
import json
import tempfile
import urllib.request
import urllib.error
from datetime import datetime, timedelta
from pathlib import Path

BASE_URL = os.environ.get("BASE_URL", "http://127.0.0.1:8090")
PB_ADMIN_EMAIL = os.environ.get("PB_ADMIN_EMAIL")
PB_ADMIN_PASSWORD = os.environ.get("PB_ADMIN_PASSWORD")

if not PB_ADMIN_EMAIL or not PB_ADMIN_PASSWORD:
    print("Error: PB_ADMIN_EMAIL and PB_ADMIN_PASSWORD environment variables are required.")
    print("Create a superuser first: ./pocketbase superuser create admin@example.com password")
    sys.exit(1)

def api_request(method, endpoint, data=None, headers=None, files=None):
    """Make an API request."""
    url = f"{BASE_URL}{endpoint}"
    headers = headers or {}

    if files:
        # Multipart form data
        import mimetypes
        boundary = "----WebKitFormBoundary7MA4YWxkTrZu0gW"
        headers["Content-Type"] = f"multipart/form-data; boundary={boundary}"

        body_parts = []
        for key, value in (data or {}).items():
            if isinstance(value, list):
                for v in value:
                    body_parts.append(f'--{boundary}\r\nContent-Disposition: form-data; name="{key}"\r\n\r\n{v}')
            else:
                body_parts.append(f'--{boundary}\r\nContent-Disposition: form-data; name="{key}"\r\n\r\n{value}')

        for key, filepath in files.items():
            filename = os.path.basename(filepath)
            mime_type = mimetypes.guess_type(filename)[0] or "application/octet-stream"
            with open(filepath, "rb") as f:
                file_data = f.read()
            body_parts.append(
                f'--{boundary}\r\nContent-Disposition: form-data; name="{key}"; filename="{filename}"\r\n'
                f'Content-Type: {mime_type}\r\n\r\n'.encode() + file_data
            )

        # Build body
        body = b""
        for part in body_parts:
            if isinstance(part, str):
                body += part.encode() + b"\r\n"
            else:
                body += part + b"\r\n"
        body += f"--{boundary}--\r\n".encode()

    elif data:
        headers["Content-Type"] = "application/json"
        body = json.dumps(data).encode()
    else:
        body = None

    req = urllib.request.Request(url, data=body, headers=headers, method=method)

    try:
        with urllib.request.urlopen(req, timeout=30) as response:
            return json.loads(response.read().decode())
    except urllib.error.HTTPError as e:
        error_body = e.read().decode() if e.fp else ""
        print(f"HTTP Error {e.code}: {error_body}")
        return None
    except Exception as e:
        print(f"Error: {e}")
        return None

def download_image(url, filepath):
    """Download an image to a file."""
    try:
        urllib.request.urlretrieve(url, filepath)
        return True
    except Exception as e:
        print(f"Failed to download {url}: {e}")
        return False

def main():
    print(f"Seeding data to {BASE_URL}...")

    # Wait for server
    for _ in range(10):
        try:
            urllib.request.urlopen(f"{BASE_URL}/api/health", timeout=2)
            break
        except:
            print("Waiting for server...")
            import time
            time.sleep(1)

    # Authenticate
    print("Authenticating as superuser...")
    auth = api_request("POST", "/api/collections/_superusers/auth-with-password", {
        "identity": PB_ADMIN_EMAIL,
        "password": PB_ADMIN_PASSWORD
    })

    if not auth or "token" not in auth:
        print("Error: Failed to authenticate as superuser")
        sys.exit(1)

    token = auth["token"]
    auth_headers = {"Authorization": token}

    # Create users
    print("Creating users...")
    users = [
        {"email": "editor@example.com", "password": "editorpassword123", "passwordConfirm": "editorpassword123", "display_name": "Editor", "role": "editor"},
        {"email": "admin@example.com", "password": "adminpassword123", "passwordConfirm": "adminpassword123", "display_name": "Admin", "role": "admin"},
        {"email": "user@example.com", "password": "userpassword123", "passwordConfirm": "userpassword123", "display_name": "Test User", "role": "user"},
    ]
    for user in users:
        api_request("POST", "/api/collections/users/records", user, auth_headers)
        print(f"  {user['role'].title()}: {user['email']} / {user['password']}")

    # Create tags
    print("Creating tags...")
    tags_data = [
        {"name": "music", "color": "#e74c3c", "status": "approved"},
        {"name": "art", "color": "#9b59b6", "status": "approved"},
        {"name": "community", "color": "#3498db", "status": "approved"},
        {"name": "workshop", "color": "#2ecc71", "status": "approved"},
        {"name": "food", "color": "#f39c12", "status": "approved"},
    ]
    tags = {}
    for tag_data in tags_data:
        result = api_request("POST", "/api/collections/tags/records", tag_data, auth_headers)
        if result and "id" in result:
            tags[tag_data["name"]] = result["id"]
            print(f"  Created tag: {tag_data['name']}")
        else:
            print(f"  Failed to create tag: {tag_data['name']}")

    if len(tags) < 5:
        print(f"Warning: Only created {len(tags)} of 5 tags. Check authentication and permissions.")

    # Create places
    print("Creating places...")
    places_data = [
        {"name": "Community Center", "address": "123 Main St", "city": "Portland", "latitude": 45.5152, "longitude": -122.6784, "status": "approved"},
        {"name": "The Art Space", "address": "456 Gallery Ave", "city": "Portland", "latitude": 45.5231, "longitude": -122.6765, "status": "approved"},
        {"name": "Riverside Park", "address": "789 River Rd", "city": "Portland", "latitude": 45.5289, "longitude": -122.6951, "status": "approved"},
    ]
    places = []
    for place_data in places_data:
        result = api_request("POST", "/api/collections/places/records", place_data, auth_headers)
        if result:
            places.append(result["id"])

    print(f"  Places: {', '.join(places)}")
    print(f"  Tags: {tags}")

    # Calculate dates
    tomorrow = (datetime.now() + timedelta(days=1)).strftime("%Y-%m-%d")
    next_week = (datetime.now() + timedelta(days=7)).strftime("%Y-%m-%d")
    in_two_weeks = (datetime.now() + timedelta(days=14)).strftime("%Y-%m-%d")
    next_month = (datetime.now() + timedelta(days=30)).strftime("%Y-%m-%d")

    # Download images
    print("Downloading event images...")
    img_dir = tempfile.mkdtemp()
    images = {
        "openmic": f"{img_dir}/openmic.jpg",
        "drawing": f"{img_dir}/drawing.jpg",
        "potluck": f"{img_dir}/potluck.jpg",
        "jazz": f"{img_dir}/jazz.jpg",
        "pottery": f"{img_dir}/pottery.jpg",
        "cleanup": f"{img_dir}/cleanup.jpg",
    }
    for name, path in images.items():
        download_image(f"https://picsum.photos/seed/{name}/400/300", path)

    # Helper to get tag IDs, filtering out None values
    def get_tags(*names):
        return [tags[n] for n in names if n in tags]

    # Create events
    print("Creating events...")
    events = [
        {
            "data": {
                "title": "Open Mic Night",
                "description": "Join us for an evening of **local talent**! Singers, poets, comedians all welcome. Sign up at the door.\n\n*Free entry, donations appreciated.*",
                "start_datetime": f"{tomorrow}T19:00:00Z",
                "end_datetime": f"{tomorrow}T22:00:00Z",
                "place": places[0] if places else "",
                "tags": get_tags("music", "community"),
                "status": "published",
            },
            "image": images["openmic"],
        },
        {
            "data": {
                "title": "Figure Drawing Workshop",
                "description": "Weekly figure drawing session with **live models**. All skill levels welcome.\n\n## What to bring\n- Your own materials, or\n- Use ours for a small fee",
                "start_datetime": f"{next_week}T14:00:00Z",
                "end_datetime": f"{next_week}T17:00:00Z",
                "place": places[1] if len(places) > 1 else "",
                "tags": get_tags("art", "workshop"),
                "status": "published",
            },
            "image": images["drawing"],
        },
        {
            "data": {
                "title": "Community Potluck",
                "description": "Monthly community potluck in the park! **Bring a dish to share** and meet your neighbors.\n\n## We provide\n- Tables and chairs\n\n## You bring\n- Your own plates and utensils",
                "start_datetime": f"{next_week}T12:00:00Z",
                "end_datetime": f"{next_week}T15:00:00Z",
                "place": places[2] if len(places) > 2 else "",
                "tags": get_tags("community", "food"),
                "status": "published",
            },
            "image": images["potluck"],
        },
        {
            "data": {
                "title": "Jazz in the Park",
                "description": "**Free jazz concert** featuring local musicians. Bring a blanket and enjoy the music!\n\nFood trucks will be on site.",
                "start_datetime": f"{in_two_weeks}T18:00:00Z",
                "end_datetime": f"{in_two_weeks}T21:00:00Z",
                "place": places[2] if len(places) > 2 else "",
                "tags": get_tags("music"),
                "status": "published",
            },
            "image": images["jazz"],
        },
        {
            "data": {
                "title": "Pottery for Beginners",
                "description": "Learn the basics of **wheel throwing** in this hands-on workshop.\n\n## Details\n- All materials provided\n- Wear clothes you don't mind getting dirty!\n\n*No experience necessary.*",
                "start_datetime": f"{next_month}T10:00:00Z",
                "end_datetime": f"{next_month}T13:00:00Z",
                "place": places[1] if len(places) > 1 else "",
                "tags": get_tags("art", "workshop"),
                "status": "published",
            },
            "image": images["pottery"],
        },
        {
            "data": {
                "title": "Neighborhood Cleanup",
                "description": "Help keep our neighborhood beautiful!\n\n## We'll provide\n- Gloves\n- Bags\n- Refreshments\n\n**Meet at the Community Center entrance.**",
                "start_datetime": f"{in_two_weeks}T09:00:00Z",
                "end_datetime": f"{in_two_weeks}T12:00:00Z",
                "place": places[0] if places else "",
                "tags": get_tags("community"),
                "status": "published",
            },
            "image": images["cleanup"],
        },
    ]

    for event in events:
        result = api_request("POST", "/api/collections/events/records", event["data"], auth_headers, {"image": event["image"]})
        if result:
            print(f"  Created: {event['data']['title']}")
        else:
            # Fallback: create without image
            result = api_request("POST", "/api/collections/events/records", event["data"], auth_headers)
            if result:
                print(f"  Created: {event['data']['title']} (without image)")
            else:
                print(f"  Failed: {event['data']['title']}")

    # Cleanup
    import shutil
    shutil.rmtree(img_dir, ignore_errors=True)

    print()
    print("Seed complete! Created 3 users, 5 tags, 3 places, and 6 events.")
    print()
    print("Frontend login (/login):")
    print("  admin@example.com  / adminpassword123  (admin)")
    print("  editor@example.com / editorpassword123 (editor)")
    print("  user@example.com   / userpassword123   (user)")

if __name__ == "__main__":
    main()
