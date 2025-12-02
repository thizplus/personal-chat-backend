# Notes API - Troubleshooting Guide

**Date:** 2025-12-02
**Issue:** Cannot POST /api/v1/api/notes
**Status:** ✅ **Resolved - URL Error**

---

## 🔴 Problem

```json
{
  "message": "Cannot POST /api/v1/api/notes",
  "success": false
}
```

**Your URL:** ❌ `https://b01.ngrok.dev/api/v1/api/notes`

---

## ✅ Solution

### URL ผิด - มี `/api` ซ้ำ 2 ครั้ง!

**❌ Wrong URL:**
```
https://b01.ngrok.dev/api/v1/api/notes
                          ^^^^ ← ซ้ำ!
```

**✅ Correct URL:**
```
https://b01.ngrok.dev/api/v1/notes
                          ^^^^^^^^ ← ถูกต้อง
```

---

## 📋 Correct API Endpoints

### Base URL
```
https://b01.ngrok.dev/api/v1
```

### All Notes Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/v1/notes` | สร้างบันทึกใหม่ |
| `GET` | `/api/v1/notes` | ดึงรายการบันทึกทั้งหมด |
| `GET` | `/api/v1/notes/:id` | ดึงบันทึกเฉพาะ |
| `PUT` | `/api/v1/notes/:id` | อัปเดตบันทึก |
| `DELETE` | `/api/v1/notes/:id` | ลบบันทึก |
| `PUT` | `/api/v1/notes/:id/pin` | ปักหมุดบันทึก |
| `DELETE` | `/api/v1/notes/:id/pin` | ยกเลิกปักหมุด |
| `GET` | `/api/v1/notes/pinned` | ดึงบันทึกที่ปักหมุด |
| `GET` | `/api/v1/notes/search?q=...` | ค้นหาบันทึก |
| `GET` | `/api/v1/notes/by-tag?tag=...` | ดึงบันทึกตาม tag |

---

## 🧪 Test with cURL

### 1. Create Note

```bash
curl -X POST https://b01.ngrok.dev/api/v1/notes \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "My First Note",
    "content": "This is a test note",
    "tags": ["test", "example"]
  }'
```

### 2. Get All Notes

```bash
curl https://b01.ngrok.dev/api/v1/notes \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 3. Search Notes

```bash
curl "https://b01.ngrok.dev/api/v1/notes/search?q=test" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

---

## 🔍 Verification Steps

### Step 1: Check Health Endpoint

```bash
curl https://b01.ngrok.dev/api/v1/health
```

**Expected Response:**
```json
{
  "status": "ok",
  "message": "API is running"
}
```

### Step 2: Check Authentication

Make sure you have valid JWT token:

```bash
# Login first
curl -X POST https://b01.ngrok.dev/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "your_username",
    "password": "your_password"
  }'
```

**Get token from response:**
```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "..."
  }
}
```

### Step 3: Use Token with Notes API

```bash
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

curl -X POST https://b01.ngrok.dev/api/v1/notes \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Test Note",
    "content": "Testing with valid token",
    "tags": []
  }'
```

---

## 📱 Frontend Integration (JavaScript/TypeScript)

### Correct Implementation

```typescript
// ✅ Correct Base URL
const BASE_URL = 'https://b01.ngrok.dev/api/v1';

// ✅ Correct Notes API
const notesApi = {
  createNote: async (data: { title: string; content: string; tags: string[] }) => {
    const response = await fetch(`${BASE_URL}/notes`, {  // ✅ /notes (not /api/notes)
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${getToken()}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(data)
    });
    return response.json();
  },

  getNotes: async (limit = 20, offset = 0) => {
    const response = await fetch(
      `${BASE_URL}/notes?limit=${limit}&offset=${offset}`,  // ✅ Correct
      {
        headers: { 'Authorization': `Bearer ${getToken()}` }
      }
    );
    return response.json();
  },

  searchNotes: async (query: string) => {
    const response = await fetch(
      `${BASE_URL}/notes/search?q=${encodeURIComponent(query)}`,  // ✅ Correct
      {
        headers: { 'Authorization': `Bearer ${getToken()}` }
      }
    );
    return response.json();
  }
};
```

### ❌ Common Mistakes

```typescript
// ❌ WRONG - Don't do this
const BASE_URL = 'https://b01.ngrok.dev/api/v1/api';  // ← /api twice!

// ❌ WRONG
fetch(`${BASE_URL}/api/notes`)  // → /api/v1/api/api/notes (3 times!)

// ❌ WRONG
fetch('https://b01.ngrok.dev/api/v1/api/notes')  // → /api twice
```

---

## 🔧 Backend Routes Configuration

Routes are correctly set up:

```go
// interfaces/api/routes/routes.go:33
api := app.Group("/api/v1")

// interfaces/api/routes/routes.go:58
SetupNoteRoutes(api, noteHandler)

// interfaces/api/routes/note_routes.go:13
notes := router.Group("/notes")  // This creates /api/v1/notes
```

**Final URL Structure:**
```
/api/v1              ← from routes.go
        /notes       ← from note_routes.go
               /     ← CreateNote handler
```

**Result:** `/api/v1/notes` ✅

---

## 🚨 Common Errors & Solutions

### Error 1: 404 Not Found

**Symptoms:**
```json
{
  "message": "Cannot POST /api/v1/api/notes",
  "success": false
}
```

**Cause:** URL มี `/api` ซ้ำ

**Solution:** ลบ `/api` ออก 1 ตัว → ใช้ `/api/v1/notes`

---

### Error 2: 401 Unauthorized

**Symptoms:**
```json
{
  "success": false,
  "message": "Unauthorized: missing or malformed JWT"
}
```

**Cause:** ไม่ได้ส่ง Authorization header หรือ token หมดอายุ

**Solution:**
```bash
# 1. Login to get new token
curl -X POST https://b01.ngrok.dev/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"your_username","password":"your_password"}'

# 2. Use new token
curl -X POST https://b01.ngrok.dev/api/v1/notes \
  -H "Authorization: Bearer NEW_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{"title":"Test","content":"Test","tags":[]}'
```

---

### Error 3: 500 Internal Server Error

**Symptoms:**
```json
{
  "success": false,
  "message": "Internal server error"
}
```

**Possible Causes:**
1. Database connection error
2. Migration not run
3. Server crashed

**Solution:**
```bash
# 1. Check server logs
tail -f logs/app.log

# 2. Check database connection
psql -h localhost -U postgres -d chat_db

# 3. Run migrations
psql -h localhost -d chat_db -U postgres -f migrations/008_create_notes.sql

# 4. Restart server
./bin/api.exe
```

---

## 📚 Documentation References

### For API Usage:
- **NOTES_APP_API.md** - Complete API documentation
- **NOTES_API_STATUS_SUMMARY.md** - Implementation status

### For Troubleshooting:
- **This file** - Troubleshooting guide

---

## ✅ Quick Checklist

Before calling Notes API:

- [ ] Check URL: `/api/v1/notes` (not `/api/v1/api/notes`)
- [ ] Have valid JWT token
- [ ] Send `Authorization: Bearer TOKEN` header
- [ ] Send `Content-Type: application/json` header
- [ ] Server is running
- [ ] Database is connected
- [ ] Migrations are run

---

## 🎯 Testing Script

Save as `test_notes_api.sh`:

```bash
#!/bin/bash

BASE_URL="https://b01.ngrok.dev/api/v1"
TOKEN="YOUR_TOKEN_HERE"

echo "1. Testing Health Check..."
curl -s "$BASE_URL/health" | jq .

echo -e "\n2. Creating Note..."
curl -s -X POST "$BASE_URL/notes" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"Test Note","content":"Testing API","tags":["test"]}' \
  | jq .

echo -e "\n3. Getting All Notes..."
curl -s "$BASE_URL/notes" \
  -H "Authorization: Bearer $TOKEN" \
  | jq .

echo -e "\n4. Searching Notes..."
curl -s "$BASE_URL/notes/search?q=test" \
  -H "Authorization: Bearer $TOKEN" \
  | jq .

echo -e "\nDone!"
```

**Usage:**
```bash
chmod +x test_notes_api.sh
./test_notes_api.sh
```

---

## 📞 Still Having Issues?

### 1. Check Server Status

```bash
# Is server running?
curl https://b01.ngrok.dev/api/v1/health

# Check ngrok status
curl http://localhost:4040/api/tunnels
```

### 2. Check Logs

```bash
# Backend logs
tail -f logs/app.log

# Nginx/Proxy logs (if any)
tail -f /var/log/nginx/error.log
```

### 3. Verify Database

```bash
# Connect to database
psql -h localhost -U postgres -d chat_db

# Check if notes table exists
\dt notes

# Check table structure
\d notes

# Check if there are any notes
SELECT COUNT(*) FROM notes;
```

---

## 🎓 Learning Points

### URL Structure in Fiber

```go
app := fiber.New()
api := app.Group("/api/v1")     // Base: /api/v1
notes := api.Group("/notes")     // Full: /api/v1/notes
notes.Post("/", handler)         // Final: /api/v1/notes
```

### Don't Add Extra `/api`!

```
❌ /api/v1/api/notes    → Wrong (api repeated)
✅ /api/v1/notes        → Correct
```

---

**Document Version:** 1.0
**Created:** 2025-12-02
**Status:** ✅ Issue Resolved
**Solution:** Use `/api/v1/notes` instead of `/api/v1/api/notes`
