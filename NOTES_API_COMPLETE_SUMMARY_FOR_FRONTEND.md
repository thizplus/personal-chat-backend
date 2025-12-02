# Notes API - Complete Summary for Frontend

**Date:** 2025-12-03
**Status:** ✅ **Ready for Integration**
**Version:** 2.0 (with Conversation Support)

---

## 🎯 TL;DR - สิ่งที่ Frontend ต้องรู้

### ✅ Features Available
1. **Personal Notes (Global)** - บันทึกส่วนตัว
2. **Conversation Notes** - บันทึกเฉพาะการสนทนา
3. **Pin/Unpin Notes** - ปักหมุดบันทึก
4. **Search** - ค้นหาด้วย full-text search
5. **Tags** - จัดกลุ่มด้วย tags
6. **Filter by Conversation** - กรองตาม conversation

### ⚠️ Breaking Changes
**None!** - Backward compatible ทั้งหมด

---

## 📋 Complete API Endpoints

### Base URL
```
https://b01.ngrok.dev/api/v1/notes
```

### All Endpoints

| Method | Endpoint | Description | New? |
|--------|----------|-------------|------|
| `POST` | `/notes` | สร้างบันทึกใหม่ (รองรับ conversation_id) | 🆕 Updated |
| `GET` | `/notes` | ดึงรายการบันทึก (รองรับ filter) | 🆕 Updated |
| `GET` | `/notes/:id` | ดึงบันทึกเฉพาะ | ✅ |
| `PUT` | `/notes/:id` | อัปเดตบันทึก | ✅ |
| `DELETE` | `/notes/:id` | ลบบันทึก | ✅ |
| `PUT` | `/notes/:id/pin` | ปักหมุดบันทึก | ✅ |
| `POST` | `/notes/:id/pin` | ปักหมุดบันทึก (alternative) | 🆕 |
| `DELETE` | `/notes/:id/pin` | ยกเลิกปักหมุด | ✅ |
| `GET` | `/notes/pinned` | ดึงบันทึกที่ปักหมุด | ✅ |
| `GET` | `/notes/search?q=...` | ค้นหาบันทึก | ✅ |
| `GET` | `/notes/by-tag?tag=...` | ดึงบันทึกตาม tag | ✅ |

---

## 🆕 What's New in v2.0

### 1. Conversation-Scoped Notes

**Before:** Notes เป็น global เท่านั้น
**Now:** รองรับทั้ง global และ conversation-scoped

```typescript
// Global Note
{
  "conversation_id": null  // Personal note
}

// Conversation Note
{
  "conversation_id": "abc-123-def"  // Linked to conversation
}
```

### 2. Query Filtering

**New Query Parameters:**
- `?conversation_id=<uuid>` - Filter by conversation
- `?scope=global` - Only global notes
- `?scope=all` - All notes (default)

### 3. Dual Method Support for Pin

**Both work:**
- `PUT /notes/:id/pin` ✅
- `POST /notes/:id/pin` ✅ (New!)

---

## 📖 API Documentation

### 1. Create Note

#### Endpoint
```
POST /api/v1/notes
```

#### Headers
```
Authorization: Bearer <token>
Content-Type: application/json
```

#### Request Body

**Global Note (Personal):**
```json
{
  "title": "Shopping List",
  "content": "Milk, Eggs, Bread",
  "tags": ["personal", "shopping"]
}
```

**Conversation Note:**
```json
{
  "conversation_id": "abc-123-def-456",  // 🆕 Optional
  "title": "Meeting Notes",
  "content": "Action items from meeting...",
  "tags": ["work", "meeting"]
}
```

#### Response (Success)
```json
{
  "success": true,
  "message": "Note created successfully",
  "data": {
    "id": "note-uuid",
    "user_id": "user-uuid",
    "conversation_id": "conv-uuid",  // or null
    "title": "Meeting Notes",
    "content": "...",
    "tags": ["work", "meeting"],
    "is_pinned": false,
    "created_at": "2025-12-03T10:00:00Z",
    "updated_at": "2025-12-03T10:00:00Z"
  }
}
```

#### Errors
```json
// 400 - Invalid conversation_id
{
  "success": false,
  "message": "Invalid conversation_id format"
}

// 403 - Not a member
{
  "success": false,
  "message": "user is not a member of this conversation"
}

// 401 - Unauthorized
{
  "success": false,
  "message": "Unauthorized: ..."
}
```

---

### 2. Get Notes (With Filters)

#### Endpoint
```
GET /api/v1/notes
```

#### Query Parameters

| Parameter | Type | Description | Example |
|-----------|------|-------------|---------|
| `conversation_id` | UUID | Filter by conversation | `?conversation_id=abc-123` |
| `scope` | String | "global" or "all" | `?scope=global` |
| `limit` | Number | Max 100, default 20 | `?limit=50` |
| `offset` | Number | Pagination offset | `?offset=0` |

#### Examples

**Get All Notes:**
```bash
GET /api/v1/notes
```

**Get Only Global Notes:**
```bash
GET /api/v1/notes?scope=global
```

**Get Conversation Notes:**
```bash
GET /api/v1/notes?conversation_id=abc-123-def
```

**Pagination:**
```bash
GET /api/v1/notes?limit=50&offset=0
```

#### Response
```json
{
  "success": true,
  "data": {
    "notes": [
      {
        "id": "note-1",
        "user_id": "user-1",
        "conversation_id": null,  // Global note
        "title": "Shopping List",
        "content": "...",
        "tags": ["personal"],
        "is_pinned": true,
        "created_at": "...",
        "updated_at": "..."
      },
      {
        "id": "note-2",
        "user_id": "user-1",
        "conversation_id": "conv-abc",  // Conversation note
        "title": "Meeting Notes",
        "content": "...",
        "tags": ["work"],
        "is_pinned": false,
        "created_at": "...",
        "updated_at": "..."
      }
    ],
    "pagination": {
      "total": 25,
      "limit": 20,
      "offset": 0
    }
  }
}
```

---

### 3. Pin Note

#### Endpoint (Both Methods Work)
```
PUT /api/v1/notes/:id/pin
POST /api/v1/notes/:id/pin  ← 🆕 Also supported
```

#### Headers
```
Authorization: Bearer <token>
```

#### Response (Success)
```json
{
  "success": true,
  "message": "Note pinned successfully"
}
```

#### Errors
```json
// 404 - Note not found
{
  "success": false,
  "message": "note not found"
}

// 400 - Already pinned
{
  "success": false,
  "message": "note is already pinned"
}
```

---

### 4. Unpin Note

#### Endpoint
```
DELETE /api/v1/notes/:id/pin
```

#### Response (Success)
```json
{
  "success": true,
  "message": "Note unpinned successfully"
}
```

#### Errors
```json
// 400 - Not pinned
{
  "success": false,
  "message": "note is not pinned"
}
```

---

### 5. Get Pinned Notes

#### Endpoint
```
GET /api/v1/notes/pinned
```

#### Query Parameters
- `limit` (optional, default: 20, max: 100)
- `offset` (optional, default: 0)

#### Response
```json
{
  "success": true,
  "data": {
    "notes": [...],  // Only pinned notes
    "pagination": {
      "total": 5,
      "limit": 20,
      "offset": 0
    }
  }
}
```

---

### 6. Search Notes

#### Endpoint
```
GET /api/v1/notes/search?q=<query>
```

#### Parameters
- `q` (required) - Search query
- `limit` (optional)
- `offset` (optional)

#### Example
```bash
GET /api/v1/notes/search?q=meeting&limit=20
```

#### Response
```json
{
  "success": true,
  "data": {
    "notes": [...],  // Matching notes
    "pagination": {...}
  }
}
```

---

### 7. Get Notes by Tag

#### Endpoint
```
GET /api/v1/notes/by-tag?tag=<tag>
```

#### Example
```bash
GET /api/v1/notes/by-tag?tag=work
```

#### Response
```json
{
  "success": true,
  "data": {
    "notes": [...],  // Notes with matching tag
    "pagination": {...}
  }
}
```

---

### 8. Update Note

#### Endpoint
```
PUT /api/v1/notes/:id
```

#### Request Body
```json
{
  "title": "Updated Title",
  "content": "Updated content...",
  "tags": ["updated", "tags"]
}
```

#### Response
```json
{
  "success": true,
  "message": "Note updated successfully",
  "data": {
    "id": "...",
    "title": "Updated Title",
    "content": "...",
    "tags": ["updated", "tags"],
    "updated_at": "2025-12-03T11:00:00Z"
  }
}
```

---

### 9. Delete Note

#### Endpoint
```
DELETE /api/v1/notes/:id
```

#### Response
```json
{
  "success": true,
  "message": "Note deleted successfully"
}
```

---

## 💻 TypeScript Integration

### Types

```typescript
interface Note {
  id: string;
  user_id: string;
  conversation_id?: string | null;  // 🆕 Optional
  title: string;
  content: string;
  tags: string[];
  is_pinned: boolean;
  created_at: string;
  updated_at: string;
}

interface CreateNoteRequest {
  conversation_id?: string;  // 🆕 Optional
  title: string;
  content: string;
  tags: string[];
}

interface NotesResponse {
  success: boolean;
  data: {
    notes: Note[];
    pagination: {
      total: number;
      limit: number;
      offset: number;
    };
  };
}

interface NoteResponse {
  success: boolean;
  message?: string;
  data?: Note;
}
```

---

### API Client

```typescript
const BASE_URL = 'https://b01.ngrok.dev/api/v1';

class NotesAPI {
  private getHeaders() {
    return {
      'Authorization': `Bearer ${getToken()}`,
      'Content-Type': 'application/json'
    };
  }

  // Create global note
  async createGlobalNote(data: Omit<CreateNoteRequest, 'conversation_id'>): Promise<Note> {
    const response = await fetch(`${BASE_URL}/notes`, {
      method: 'POST',
      headers: this.getHeaders(),
      body: JSON.stringify(data)
    });
    const result = await response.json();
    return result.data;
  }

  // Create conversation note
  async createConversationNote(
    conversationId: string,
    data: Omit<CreateNoteRequest, 'conversation_id'>
  ): Promise<Note> {
    const response = await fetch(`${BASE_URL}/notes`, {
      method: 'POST',
      headers: this.getHeaders(),
      body: JSON.stringify({
        ...data,
        conversation_id: conversationId
      })
    });
    const result = await response.json();
    return result.data;
  }

  // Get all notes
  async getAllNotes(limit = 20, offset = 0): Promise<NotesResponse> {
    const response = await fetch(
      `${BASE_URL}/notes?limit=${limit}&offset=${offset}`,
      { headers: this.getHeaders() }
    );
    return response.json();
  }

  // Get global notes only
  async getGlobalNotes(limit = 20, offset = 0): Promise<NotesResponse> {
    const response = await fetch(
      `${BASE_URL}/notes?scope=global&limit=${limit}&offset=${offset}`,
      { headers: this.getHeaders() }
    );
    return response.json();
  }

  // Get conversation notes
  async getConversationNotes(
    conversationId: string,
    limit = 20,
    offset = 0
  ): Promise<NotesResponse> {
    const response = await fetch(
      `${BASE_URL}/notes?conversation_id=${conversationId}&limit=${limit}&offset=${offset}`,
      { headers: this.getHeaders() }
    );
    return response.json();
  }

  // Pin note (supports both PUT and POST)
  async pinNote(noteId: string): Promise<void> {
    await fetch(`${BASE_URL}/notes/${noteId}/pin`, {
      method: 'POST',  // or 'PUT', both work!
      headers: this.getHeaders()
    });
  }

  // Unpin note
  async unpinNote(noteId: string): Promise<void> {
    await fetch(`${BASE_URL}/notes/${noteId}/pin`, {
      method: 'DELETE',
      headers: this.getHeaders()
    });
  }

  // Get pinned notes
  async getPinnedNotes(limit = 20, offset = 0): Promise<NotesResponse> {
    const response = await fetch(
      `${BASE_URL}/notes/pinned?limit=${limit}&offset=${offset}`,
      { headers: this.getHeaders() }
    );
    return response.json();
  }

  // Search notes
  async searchNotes(query: string, limit = 20, offset = 0): Promise<NotesResponse> {
    const response = await fetch(
      `${BASE_URL}/notes/search?q=${encodeURIComponent(query)}&limit=${limit}&offset=${offset}`,
      { headers: this.getHeaders() }
    );
    return response.json();
  }

  // Get notes by tag
  async getNotesByTag(tag: string, limit = 20, offset = 0): Promise<NotesResponse> {
    const response = await fetch(
      `${BASE_URL}/notes/by-tag?tag=${encodeURIComponent(tag)}&limit=${limit}&offset=${offset}`,
      { headers: this.getHeaders() }
    );
    return response.json();
  }

  // Update note
  async updateNote(noteId: string, data: Omit<CreateNoteRequest, 'conversation_id'>): Promise<Note> {
    const response = await fetch(`${BASE_URL}/notes/${noteId}`, {
      method: 'PUT',
      headers: this.getHeaders(),
      body: JSON.stringify(data)
    });
    const result = await response.json();
    return result.data;
  }

  // Delete note
  async deleteNote(noteId: string): Promise<void> {
    await fetch(`${BASE_URL}/notes/${noteId}`, {
      method: 'DELETE',
      headers: this.getHeaders()
    });
  }
}

export const notesAPI = new NotesAPI();
```

---

### React Hooks (Optional)

```typescript
// useNotes.ts
import { useState, useEffect } from 'react';
import { notesAPI } from './notesAPI';

export function useNotes(conversationId?: string) {
  const [notes, setNotes] = useState<Note[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function fetchNotes() {
      setLoading(true);
      try {
        const response = conversationId
          ? await notesAPI.getConversationNotes(conversationId)
          : await notesAPI.getAllNotes();
        setNotes(response.data.notes);
      } catch (error) {
        console.error('Failed to fetch notes:', error);
      } finally {
        setLoading(false);
      }
    }
    fetchNotes();
  }, [conversationId]);

  return { notes, loading, refetch: () => fetchNotes() };
}

// useGlobalNotes.ts
export function useGlobalNotes() {
  const [notes, setNotes] = useState<Note[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function fetchNotes() {
      setLoading(true);
      try {
        const response = await notesAPI.getGlobalNotes();
        setNotes(response.data.notes);
      } catch (error) {
        console.error('Failed to fetch global notes:', error);
      } finally {
        setLoading(false);
      }
    }
    fetchNotes();
  }, []);

  return { notes, loading };
}
```

---

## 🎨 UI Examples

### Global Notes View

```typescript
// GlobalNotesPage.tsx
const GlobalNotesPage = () => {
  const { notes, loading } = useGlobalNotes();

  return (
    <div className="global-notes-page">
      <h1>My Personal Notes</h1>
      {loading ? (
        <Spinner />
      ) : (
        <div className="notes-grid">
          {notes.map(note => (
            <NoteCard key={note.id} note={note} />
          ))}
        </div>
      )}
      <button onClick={() => openCreateNoteModal()}>
        + Create Note
      </button>
    </div>
  );
};
```

### Conversation Notes Sidebar

```typescript
// ConversationNotesSidebar.tsx
const ConversationNotesSidebar = ({ conversationId }) => {
  const { notes, loading, refetch } = useNotes(conversationId);

  const handleCreateNote = async (data) => {
    await notesAPI.createConversationNote(conversationId, data);
    refetch();
  };

  return (
    <div className="conversation-notes-sidebar">
      <h3>Notes for this conversation</h3>
      {notes.map(note => (
        <NoteCard key={note.id} note={note} compact />
      ))}
      <button onClick={() => openCreateNoteModal(handleCreateNote)}>
        + Add Note
      </button>
    </div>
  );
};
```

---

## ⚠️ Important Notes

### 1. Permission Rules

**Global Notes:**
- Only accessible by owner
- No permission checks needed

**Conversation Notes:**
- Only accessible by conversation members
- Backend validates membership automatically
- Returns `403 Forbidden` if not a member

### 2. Privacy

- **Notes are PRIVATE** to the user who created them
- NOT shared with other conversation members
- Like "personal notes about a conversation"

### 3. Cascade Delete

- When conversation is deleted → conversation notes are deleted
- When user leaves conversation → notes remain (user's private data)

### 4. HTTP Methods

**Pin Note supports BOTH:**
- `PUT /notes/:id/pin` ✅
- `POST /notes/:id/pin` ✅

**Choose what works best for your frontend!**

---

## 🧪 Testing Checklist

### Backend Ready ✅

- [x] All endpoints implemented
- [x] Permission checks working
- [x] Route ordering fixed
- [x] Build successful
- [x] Documentation complete

### Frontend TODO

- [ ] Implement NotesAPI client
- [ ] Create Note components
- [ ] Add to Global Notes page
- [ ] Add to Conversation sidebar
- [ ] Test create global note
- [ ] Test create conversation note
- [ ] Test pin/unpin
- [ ] Test search
- [ ] Test filtering

---

## 🚀 Deployment Status

**Backend:**
- ✅ Code complete
- ✅ Build successful
- ✅ Ready to deploy

**Migration:**
```sql
-- Run this on production database
\i migrations/009_add_conversation_to_notes.sql
```

**Restart Required:** Yes (after migration)

---

## 📞 Support & Questions

**Documentation Files:**
1. `NOTES_API_COMPLETE_SUMMARY_FOR_FRONTEND.md` - This file
2. `NOTES_CONVERSATION_IMPLEMENTATION_SUMMARY.md` - Technical details
3. `NOTES_ROUTE_ORDERING_FIX.md` - Route fix explanation
4. `NOTES_CONVERSATION_DESIGN_TH.md` - Design decisions (Thai)

**Common Issues:**
1. "Method Not Allowed" → Use PUT or POST for pin
2. "403 Forbidden" → User not member of conversation
3. "Invalid conversation_id" → Check UUID format

---

## ✅ Summary

### What's Available
- ✅ Personal (Global) Notes
- ✅ Conversation Notes
- ✅ Pin/Unpin
- ✅ Search
- ✅ Tags
- ✅ Filtering

### What's New in v2.0
- 🆕 `conversation_id` field (optional)
- 🆕 Filter by conversation
- 🆕 Filter by scope (global/all)
- 🆕 POST support for pin endpoint

### Breaking Changes
- ❌ None! Fully backward compatible

**Backend is ready for frontend integration!** 🎉

---

**Created:** 2025-12-03
**Version:** 2.0
**Status:** ✅ Production Ready
