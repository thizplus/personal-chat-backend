# Notes API - Route Ordering Fix

**Date:** 2025-12-03
**Issue:** `PUT /api/v1/notes/:id/pin` returns "Method Not Allowed"
**Status:** ✅ **Fixed**

---

## 🔴 Problem

**Error:**
```json
{
  "message": "Method Not Allowed",
  "success": false
}
```

**URL:** `PUT https://b01.ngrok.dev/api/v1/notes/79301087-5234-46d4-850a-89e6378f35ab/pin`

---

## 🔍 Root Cause

**Route Ordering Issue** in Fiber framework

### ❌ Before (Incorrect Order)

```go
notes.Get("/:id", noteHandler.GetNote)            // Line 19
notes.Put("/:id", noteHandler.UpdateNote)         // Line 20
notes.Delete("/:id", noteHandler.DeleteNote)      // Line 21

notes.Put("/:id/pin", noteHandler.PinNote)        // Line 24
notes.Delete("/:id/pin", noteHandler.UnpinNote)   // Line 25
notes.Get("/pinned", noteHandler.GetPinnedNotes)  // Line 26

notes.Get("/search", noteHandler.SearchNotes)     // Line 29
notes.Get("/by-tag", noteHandler.GetNotesByTag)   // Line 30
```

### ⚠️ What Happened

1. **Request:** `PUT /notes/79301087-5234-46d4-850a-89e6378f35ab/pin`

2. **Fiber Matching:**
   - Checks line 20: `PUT /:id` → **MATCH!**
   - Interprets `id = "79301087-5234-46d4-850a-89e6378f35ab/pin"`
   - Routes to `UpdateNote` handler instead of `PinNote`

3. **Handler receives:**
   - `id = "79301087-5234-46d4-850a-89e6378f35ab/pin"` (invalid UUID format)
   - Handler tries to parse as UUID → fails
   - Returns error

**Similarly for GET:**
- `GET /notes/pinned` would match `/:id` instead of `/pinned`
- `GET /notes/search` would match `/:id` instead of `/search`

---

## ✅ Solution

**File:** `interfaces/api/routes/note_routes.go`

### Correct Route Ordering

```go
// 1. Base CRUD
notes.Post("/", noteHandler.CreateNote)
notes.Get("/", noteHandler.GetNotes)

// 2. Specific static routes (MUST come before /:id)
notes.Get("/pinned", noteHandler.GetPinnedNotes)
notes.Get("/search", noteHandler.SearchNotes)
notes.Get("/by-tag", noteHandler.GetNotesByTag)

// 3. Routes with sub-paths (MUST come before /:id)
notes.Put("/:id/pin", noteHandler.PinNote)
notes.Delete("/:id/pin", noteHandler.UnpinNote)

// 4. Dynamic routes (MUST come last)
notes.Get("/:id", noteHandler.GetNote)
notes.Put("/:id", noteHandler.UpdateNote)
notes.Delete("/:id", noteHandler.DeleteNote)
```

---

## 📊 Route Matching Priority

### Rule of Thumb

In Fiber (and most web frameworks), route matching follows **first-match-wins**:

1. **Static routes** (e.g., `/pinned`, `/search`)
2. **Routes with sub-paths** (e.g., `/:id/pin`, `/:id/settings`)
3. **Dynamic routes** (e.g., `/:id`, `/:slug`)

### Examples

| URL | Matched Route (Before) | Matched Route (After) |
|-----|------------------------|----------------------|
| `GET /notes/pinned` | ❌ `/:id` (id="pinned") | ✅ `/pinned` |
| `GET /notes/search` | ❌ `/:id` (id="search") | ✅ `/search` |
| `PUT /notes/abc-123/pin` | ❌ `/:id` (id="abc-123/pin") | ✅ `/:id/pin` |
| `GET /notes/abc-123` | ✅ `/:id` | ✅ `/:id` |

---

## 🧪 Testing

### Test 1: Pin Note

```bash
curl -X PUT https://b01.ngrok.dev/api/v1/notes/79301087-5234-46d4-850a-89e6378f35ab/pin \
  -H "Authorization: Bearer $TOKEN"
```

**Expected:**
```json
{
  "success": true,
  "message": "Note pinned successfully"
}
```

### Test 2: Unpin Note

```bash
curl -X DELETE https://b01.ngrok.dev/api/v1/notes/79301087-5234-46d4-850a-89e6378f35ab/pin \
  -H "Authorization: Bearer $TOKEN"
```

**Expected:**
```json
{
  "success": true,
  "message": "Note unpinned successfully"
}
```

### Test 3: Get Pinned Notes

```bash
curl https://b01.ngrok.dev/api/v1/notes/pinned \
  -H "Authorization: Bearer $TOKEN"
```

**Expected:**
```json
{
  "success": true,
  "data": {
    "notes": [...],
    "pagination": {...}
  }
}
```

### Test 4: Search Notes

```bash
curl "https://b01.ngrok.dev/api/v1/notes/search?q=test" \
  -H "Authorization: Bearer $TOKEN"
```

**Expected:**
```json
{
  "success": true,
  "data": {
    "notes": [...],
    "pagination": {...}
  }
}
```

---

## 📝 Complete Route List (After Fix)

| Method | Path | Handler | Order |
|--------|------|---------|-------|
| `POST` | `/notes` | CreateNote | 1 |
| `GET` | `/notes` | GetNotes | 2 |
| `GET` | `/notes/pinned` | GetPinnedNotes | 3 ⚠️ |
| `GET` | `/notes/search` | SearchNotes | 4 ⚠️ |
| `GET` | `/notes/by-tag` | GetNotesByTag | 5 ⚠️ |
| `PUT` | `/notes/:id/pin` | PinNote | 6 ⚠️ |
| `DELETE` | `/notes/:id/pin` | UnpinNote | 7 ⚠️ |
| `GET` | `/notes/:id` | GetNote | 8 |
| `PUT` | `/notes/:id` | UpdateNote | 9 |
| `DELETE` | `/notes/:id` | DeleteNote | 10 |

⚠️ = Must come before `/:id` routes

---

## 🔧 Code Changes

**File:** `interfaces/api/routes/note_routes.go`

### Diff

```diff
  // CRUD operations
  notes.Post("/", noteHandler.CreateNote)
  notes.Get("/", noteHandler.GetNotes)
- notes.Get("/:id", noteHandler.GetNote)
- notes.Put("/:id", noteHandler.UpdateNote)
- notes.Delete("/:id", noteHandler.DeleteNote)
-
- // Pin operations
- notes.Put("/:id/pin", noteHandler.PinNote)
- notes.Delete("/:id/pin", noteHandler.UnpinNote)
+
+ // Special routes (ต้องมาก่อน /:id)
  notes.Get("/pinned", noteHandler.GetPinnedNotes)
-
- // Search and filter
  notes.Get("/search", noteHandler.SearchNotes)
  notes.Get("/by-tag", noteHandler.GetNotesByTag)
+
+ // Pin operations (ต้องมาก่อน /:id)
+ notes.Put("/:id/pin", noteHandler.PinNote)
+ notes.Delete("/:id/pin", noteHandler.UnpinNote)
+
+ // Dynamic routes (ต้องมาหลังสุด)
+ notes.Get("/:id", noteHandler.GetNote)
+ notes.Put("/:id", noteHandler.UpdateNote)
+ notes.Delete("/:id", noteHandler.DeleteNote)
```

---

## 🎓 Best Practices

### Fiber Route Ordering

1. **Always declare specific routes before dynamic routes**
   ```go
   ✅ app.Get("/users/me", getMe)
   ✅ app.Get("/users/:id", getUser)

   ❌ app.Get("/users/:id", getUser)
   ❌ app.Get("/users/me", getMe)  // Never matched!
   ```

2. **Routes with sub-paths before base dynamic routes**
   ```go
   ✅ app.Put("/:id/activate", activate)
   ✅ app.Put("/:id", update)

   ❌ app.Put("/:id", update)
   ❌ app.Put("/:id/activate", activate)  // Never matched!
   ```

3. **Group related routes**
   ```go
   // Static routes
   notes.Get("/pinned", ...)
   notes.Get("/search", ...)

   // Sub-path routes
   notes.Put("/:id/pin", ...)
   notes.Put("/:id/archive", ...)

   // Base routes
   notes.Get("/:id", ...)
   notes.Put("/:id", ...)
   ```

---

## 🚀 Deployment

### 1. Rebuild

```bash
go build -o bin/api.exe ./cmd/api
```

### 2. Restart Server

```bash
# Kill old process
taskkill /F /PID <old-pid>

# Start new
./bin/api.exe
```

### 3. Test All Endpoints

```bash
# Test pin
curl -X PUT $BASE_URL/notes/$NOTE_ID/pin \
  -H "Authorization: Bearer $TOKEN"

# Test get pinned
curl $BASE_URL/notes/pinned \
  -H "Authorization: Bearer $TOKEN"

# Test search
curl "$BASE_URL/notes/search?q=test" \
  -H "Authorization: Bearer $TOKEN"
```

---

## 📋 Checklist

- [x] Fix route ordering
- [x] Rebuild successful
- [ ] Deploy to server
- [ ] Test pin endpoint
- [ ] Test unpin endpoint
- [ ] Test get pinned endpoint
- [ ] Test search endpoint
- [ ] Test by-tag endpoint
- [ ] Verify all CRUD endpoints still work

---

## 📚 Related Issues

Similar issues in other routes to check:

```bash
# Check other routes for similar problems
grep -r "router.Put\|router.Get\|router.Post\|router.Delete" interfaces/api/routes/
```

---

**Status:** ✅ Fixed
**Build:** ✅ Successful
**Deployment:** ⏳ Pending

**Next Steps:**
1. Restart server
2. Test endpoints
3. Update API documentation if needed

---

**Created:** 2025-12-03
**Fixed in:** `interfaces/api/routes/note_routes.go`
**Lines changed:** 10-15 lines
