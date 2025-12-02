# Notes App API - Backend Implementation Guide

**Status:** ✅ **COMPLETE & READY TO USE**
**Last Updated:** 2025-12-01
**Backend Version:** v2

---

## 📋 Overview

ระบบ Notes (บันทึกส่วนตัว/Memo) ช่วยให้ผู้ใช้สามารถสร้างบันทึกส่วนตัวได้ เหมือนกับแอปพลิเคชัน Notes ใน iOS/Android โดยแต่ละ Note สามารถมี title, content และ tags ได้

### Features
- ✅ สร้าง/แก้ไข/ลบบันทึกส่วนตัว
- ✅ เพิ่ม title, content และ tags
- ✅ ปักหมุด (pin) บันทึกที่สำคัญ
- ✅ ค้นหาบันทึก (full-text search)
- ✅ กรองบันทึกตาม tags
- ✅ ดูรายการบันทึกทั้งหมด
- ✅ ดูรายการบันทึกที่ปักหมุด
- ✅ Pagination support

---

## 🔗 API Endpoints Summary

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/v1/notes` | สร้างบันทึกใหม่ |
| `GET` | `/api/v1/notes` | ดึงรายการบันทึกทั้งหมด |
| `GET` | `/api/v1/notes/:id` | ดึงบันทึกเฉพาะ |
| `PUT` | `/api/v1/notes/:id` | อัปเดตบันทึก |
| `DELETE` | `/api/v1/notes/:id` | ลบบันทึก |
| `PUT` | `/api/v1/notes/:id/pin` | ปักหมุดบันทึก |
| `DELETE` | `/api/v1/notes/:id/pin` | ยกเลิกการปักหมุด |
| `GET` | `/api/v1/notes/pinned` | ดึงรายการบันทึกที่ปักหมุด |
| `GET` | `/api/v1/notes/search?q=...` | ค้นหาบันทึก |
| `GET` | `/api/v1/notes/by-tag?tag=...` | ดึงบันทึกตาม tag |

**Authentication Required:** ✅ Yes (Bearer Token) สำหรับทุก endpoint

---

## 📝 Data Model

### Note Object

```typescript
interface Note {
  id: string;                    // UUID
  user_id: string;               // UUID ของเจ้าของ note
  title: string;                 // หัวข้อ (max 255 chars)
  content: string;               // เนื้อหา (unlimited)
  tags: string[];                // array ของ tags
  is_pinned: boolean;            // ปักหมุดหรือไม่
  created_at: string;            // ISO 8601 timestamp
  updated_at: string;            // ISO 8601 timestamp
}
```

### Example Note

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440001",
  "user_id": "650e8400-e29b-41d4-a716-446655440001",
  "title": "Meeting Notes",
  "content": "Discussed Q4 roadmap:\n- Feature A\n- Feature B\n- Bug fixes",
  "tags": ["work", "meeting", "q4"],
  "is_pinned": true,
  "created_at": "2025-01-15T10:30:00Z",
  "updated_at": "2025-01-15T11:00:00Z"
}
```

---

## 📡 API Details

### 1. สร้างบันทึกใหม่

**POST** `/api/v1/notes`

#### Request

```json
{
  "title": "My First Note",
  "content": "This is the content of my note...",
  "tags": ["personal", "important"]
}
```

#### Parameters

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `title` | `string` | ❌ No | หัวข้อบันทึก (ถ้าไม่ใส่จะเป็น empty string) |
| `content` | `string` | ❌ No | เนื้อหาบันทึก |
| `tags` | `array<string>` | ❌ No | รายการ tags (default: []) |

#### Success Response (201 Created)

```json
{
  "success": true,
  "message": "Note created successfully",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440001",
    "user_id": "650e8400-e29b-41d4-a716-446655440001",
    "title": "My First Note",
    "content": "This is the content of my note...",
    "tags": ["personal", "important"],
    "is_pinned": false,
    "created_at": "2025-01-15T10:30:00Z",
    "updated_at": "2025-01-15T10:30:00Z"
  }
}
```

---

### 2. ดึงรายการบันทึกทั้งหมด

**GET** `/api/v1/notes?limit=20&offset=0`

#### Query Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `limit` | `integer` | 20 | จำนวนบันทึกต่อหน้า (max: 100) |
| `offset` | `integer` | 0 | ข้ามกี่รายการ (สำหรับ pagination) |

#### Success Response (200 OK)

```json
{
  "success": true,
  "data": {
    "notes": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440001",
        "title": "Meeting Notes",
        "content": "...",
        "tags": ["work", "meeting"],
        "is_pinned": true,
        "created_at": "2025-01-15T10:30:00Z",
        "updated_at": "2025-01-15T11:00:00Z"
      },
      {
        "id": "550e8400-e29b-41d4-a716-446655440002",
        "title": "Shopping List",
        "content": "...",
        "tags": ["personal"],
        "is_pinned": false,
        "created_at": "2025-01-14T09:00:00Z",
        "updated_at": "2025-01-14T09:00:00Z"
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

### 3. ดึงบันทึกเฉพาะ

**GET** `/api/v1/notes/:id`

#### Success Response (200 OK)

```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440001",
    "user_id": "650e8400-e29b-41d4-a716-446655440001",
    "title": "Meeting Notes",
    "content": "Full content here...",
    "tags": ["work", "meeting"],
    "is_pinned": true,
    "created_at": "2025-01-15T10:30:00Z",
    "updated_at": "2025-01-15T11:00:00Z"
  }
}
```

#### Error Response (404 Not Found)

```json
{
  "success": false,
  "message": "note not found"
}
```

---

### 4. อัปเดตบันทึก

**PUT** `/api/v1/notes/:id`

#### Request

```json
{
  "title": "Updated Title",
  "content": "Updated content...",
  "tags": ["work", "meeting", "important"]
}
```

#### Parameters

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `title` | `string` | ❌ No | หัวข้อใหม่ |
| `content` | `string` | ❌ No | เนื้อหาใหม่ |
| `tags` | `array<string>` | ❌ No | tags ใหม่ (จะ replace tags เดิมทั้งหมด) |

#### Success Response (200 OK)

```json
{
  "success": true,
  "message": "Note updated successfully",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440001",
    "title": "Updated Title",
    "content": "Updated content...",
    "tags": ["work", "meeting", "important"],
    "is_pinned": true,
    "created_at": "2025-01-15T10:30:00Z",
    "updated_at": "2025-01-15T12:00:00Z"
  }
}
```

---

### 5. ลบบันทึก

**DELETE** `/api/v1/notes/:id`

#### Success Response (200 OK)

```json
{
  "success": true,
  "message": "Note deleted successfully"
}
```

#### Error Response (404 Not Found)

```json
{
  "success": false,
  "message": "note not found"
}
```

---

### 6. ปักหมุดบันทึก

**PUT** `/api/v1/notes/:id/pin`

#### Success Response (200 OK)

```json
{
  "success": true,
  "message": "Note pinned successfully"
}
```

#### Error Response (400 Bad Request)

```json
{
  "success": false,
  "message": "note is already pinned"
}
```

---

### 7. ยกเลิกการปักหมุด

**DELETE** `/api/v1/notes/:id/pin`

#### Success Response (200 OK)

```json
{
  "success": true,
  "message": "Note unpinned successfully"
}
```

#### Error Response (400 Bad Request)

```json
{
  "success": false,
  "message": "note is not pinned"
}
```

---

### 8. ดึงรายการบันทึกที่ปักหมุด

**GET** `/api/v1/notes/pinned?limit=20&offset=0`

#### Query Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `limit` | `integer` | 20 | จำนวนบันทึกต่อหน้า (max: 100) |
| `offset` | `integer` | 0 | ข้ามกี่รายการ |

#### Success Response (200 OK)

```json
{
  "success": true,
  "data": {
    "notes": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440001",
        "title": "Important Meeting",
        "is_pinned": true,
        "created_at": "2025-01-15T10:30:00Z"
      }
    ],
    "pagination": {
      "total": 3,
      "limit": 20,
      "offset": 0
    }
  }
}
```

---

### 9. ค้นหาบันทึก

**GET** `/api/v1/notes/search?q=meeting&limit=20&offset=0`

#### Query Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `q` | `string` | ✅ Yes | คำค้นหา (ค้นหาใน title และ content) |
| `limit` | `integer` | ❌ No | จำนวนผลลัพธ์ (default: 20, max: 100) |
| `offset` | `integer` | ❌ No | ข้ามกี่รายการ (default: 0) |

#### Success Response (200 OK)

```json
{
  "success": true,
  "data": {
    "notes": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440001",
        "title": "Meeting Notes",
        "content": "Team meeting discussion...",
        "tags": ["work", "meeting"],
        "created_at": "2025-01-15T10:30:00Z"
      }
    ],
    "query": "meeting",
    "pagination": {
      "total": 5,
      "limit": 20,
      "offset": 0
    }
  }
}
```

#### Error Response (400 Bad Request)

```json
{
  "success": false,
  "message": "Search query (q) is required"
}
```

---

### 10. ดึงบันทึกตาม Tag

**GET** `/api/v1/notes/by-tag?tag=work&limit=20&offset=0`

#### Query Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `tag` | `string` | ✅ Yes | ชื่อ tag ที่ต้องการ filter |
| `limit` | `integer` | ❌ No | จำนวนผลลัพธ์ (default: 20, max: 100) |
| `offset` | `integer` | ❌ No | ข้ามกี่รายการ (default: 0) |

#### Success Response (200 OK)

```json
{
  "success": true,
  "data": {
    "notes": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440001",
        "title": "Project Planning",
        "tags": ["work", "planning"],
        "created_at": "2025-01-15T10:30:00Z"
      },
      {
        "id": "550e8400-e29b-41d4-a716-446655440002",
        "title": "Meeting Notes",
        "tags": ["work", "meeting"],
        "created_at": "2025-01-14T09:00:00Z"
      }
    ],
    "tag": "work",
    "pagination": {
      "total": 8,
      "limit": 20,
      "offset": 0
    }
  }
}
```

#### Error Response (400 Bad Request)

```json
{
  "success": false,
  "message": "Tag query parameter is required"
}
```

---

## ❌ Common Error Responses

### 401 Unauthorized
```json
{
  "success": false,
  "message": "Unauthorized: invalid or expired token"
}
```

### 404 Not Found
```json
{
  "success": false,
  "message": "note not found"
}
```

### 500 Internal Server Error
```json
{
  "success": false,
  "message": "Internal server error"
}
```

---

## 🎨 Frontend Integration Guide

### 1. React/TypeScript Example

#### Type Definitions

```typescript
interface Note {
  id: string;
  user_id: string;
  title: string;
  content: string;
  tags: string[];
  is_pinned: boolean;
  created_at: string;
  updated_at: string;
}

interface NotesState {
  notes: Note[];
  pinnedNotes: Note[];
  currentNote: Note | null;
  isLoading: boolean;
  error: string | null;
  pagination: {
    total: number;
    limit: number;
    offset: number;
  };
}
```

#### API Service

```typescript
// services/notesApi.ts
const BASE_URL = '/api/v1/notes';

export const notesApi = {
  // สร้างบันทึกใหม่
  createNote: async (data: { title: string; content: string; tags: string[] }) => {
    const response = await fetch(BASE_URL, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${getToken()}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(data)
    });
    return response.json();
  },

  // ดึงรายการบันทึก
  getNotes: async (limit = 20, offset = 0) => {
    const response = await fetch(
      `${BASE_URL}?limit=${limit}&offset=${offset}`,
      {
        headers: { 'Authorization': `Bearer ${getToken()}` }
      }
    );
    return response.json();
  },

  // ดึงบันทึกเฉพาะ
  getNote: async (id: string) => {
    const response = await fetch(`${BASE_URL}/${id}`, {
      headers: { 'Authorization': `Bearer ${getToken()}` }
    });
    return response.json();
  },

  // อัปเดตบันทึก
  updateNote: async (id: string, data: Partial<Note>) => {
    const response = await fetch(`${BASE_URL}/${id}`, {
      method: 'PUT',
      headers: {
        'Authorization': `Bearer ${getToken()}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(data)
    });
    return response.json();
  },

  // ลบบันทึก
  deleteNote: async (id: string) => {
    const response = await fetch(`${BASE_URL}/${id}`, {
      method: 'DELETE',
      headers: { 'Authorization': `Bearer ${getToken()}` }
    });
    return response.json();
  },

  // ปักหมุด/ยกเลิกปักหมุด
  togglePin: async (id: string, isPinned: boolean) => {
    const response = await fetch(`${BASE_URL}/${id}/pin`, {
      method: isPinned ? 'DELETE' : 'PUT',
      headers: { 'Authorization': `Bearer ${getToken()}` }
    });
    return response.json();
  },

  // ดึงบันทึกที่ปักหมุด
  getPinnedNotes: async (limit = 20, offset = 0) => {
    const response = await fetch(
      `${BASE_URL}/pinned?limit=${limit}&offset=${offset}`,
      {
        headers: { 'Authorization': `Bearer ${getToken()}` }
      }
    );
    return response.json();
  },

  // ค้นหาบันทึก
  searchNotes: async (query: string, limit = 20, offset = 0) => {
    const response = await fetch(
      `${BASE_URL}/search?q=${encodeURIComponent(query)}&limit=${limit}&offset=${offset}`,
      {
        headers: { 'Authorization': `Bearer ${getToken()}` }
      }
    );
    return response.json();
  },

  // ดึงบันทึกตาม tag
  getNotesByTag: async (tag: string, limit = 20, offset = 0) => {
    const response = await fetch(
      `${BASE_URL}/by-tag?tag=${encodeURIComponent(tag)}&limit=${limit}&offset=${offset}`,
      {
        headers: { 'Authorization': `Bearer ${getToken()}` }
      }
    );
    return response.json();
  }
};
```

#### Custom Hook

```typescript
// hooks/useNotes.ts
import { useState, useEffect } from 'react';
import { notesApi } from '../services/notesApi';

export const useNotes = () => {
  const [state, setState] = useState<NotesState>({
    notes: [],
    pinnedNotes: [],
    currentNote: null,
    isLoading: false,
    error: null,
    pagination: { total: 0, limit: 20, offset: 0 }
  });

  const loadNotes = async (limit = 20, offset = 0) => {
    setState(prev => ({ ...prev, isLoading: true, error: null }));
    try {
      const result = await notesApi.getNotes(limit, offset);
      if (result.success) {
        setState(prev => ({
          ...prev,
          notes: result.data.notes,
          pagination: result.data.pagination,
          isLoading: false
        }));
      }
    } catch (error) {
      setState(prev => ({
        ...prev,
        error: 'Failed to load notes',
        isLoading: false
      }));
    }
  };

  const createNote = async (data: { title: string; content: string; tags: string[] }) => {
    setState(prev => ({ ...prev, isLoading: true, error: null }));
    try {
      const result = await notesApi.createNote(data);
      if (result.success) {
        // Reload notes
        await loadNotes();
        return result.data;
      }
    } catch (error) {
      setState(prev => ({
        ...prev,
        error: 'Failed to create note',
        isLoading: false
      }));
    }
  };

  const deleteNote = async (id: string) => {
    try {
      const result = await notesApi.deleteNote(id);
      if (result.success) {
        // Remove from local state
        setState(prev => ({
          ...prev,
          notes: prev.notes.filter(note => note.id !== id),
          pinnedNotes: prev.pinnedNotes.filter(note => note.id !== id)
        }));
      }
    } catch (error) {
      setState(prev => ({
        ...prev,
        error: 'Failed to delete note'
      }));
    }
  };

  const togglePin = async (id: string, isPinned: boolean) => {
    try {
      const result = await notesApi.togglePin(id, isPinned);
      if (result.success) {
        // Update local state
        setState(prev => ({
          ...prev,
          notes: prev.notes.map(note =>
            note.id === id ? { ...note, is_pinned: !isPinned } : note
          )
        }));
      }
    } catch (error) {
      setState(prev => ({
        ...prev,
        error: 'Failed to toggle pin'
      }));
    }
  };

  return {
    ...state,
    loadNotes,
    createNote,
    deleteNote,
    togglePin
  };
};
```

---

### 2. UI Components Example

#### Note List

```typescript
const NotesList: React.FC = () => {
  const { notes, isLoading, loadNotes, deleteNote, togglePin } = useNotes();

  useEffect(() => {
    loadNotes();
  }, []);

  if (isLoading) return <div>Loading...</div>;

  return (
    <div className="notes-list">
      {notes.map(note => (
        <NoteCard
          key={note.id}
          note={note}
          onDelete={deleteNote}
          onTogglePin={togglePin}
        />
      ))}
    </div>
  );
};
```

#### Note Card

```typescript
const NoteCard: React.FC<{
  note: Note;
  onDelete: (id: string) => void;
  onTogglePin: (id: string, isPinned: boolean) => void;
}> = ({ note, onDelete, onTogglePin }) => {
  return (
    <div className="note-card">
      <div className="note-header">
        <h3>{note.title || 'Untitled'}</h3>
        <button onClick={() => onTogglePin(note.id, note.is_pinned)}>
          {note.is_pinned ? '📌 Pinned' : 'Pin'}
        </button>
      </div>

      <p className="note-content">
        {note.content.substring(0, 100)}...
      </p>

      <div className="note-tags">
        {note.tags.map(tag => (
          <span key={tag} className="tag">#{tag}</span>
        ))}
      </div>

      <div className="note-footer">
        <span>{new Date(note.created_at).toLocaleDateString()}</span>
        <button onClick={() => onDelete(note.id)}>Delete</button>
      </div>
    </div>
  );
};
```

#### Create/Edit Form

```typescript
const NoteForm: React.FC<{ noteId?: string }> = ({ noteId }) => {
  const [formData, setFormData] = useState({
    title: '',
    content: '',
    tags: [] as string[]
  });

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (noteId) {
      await notesApi.updateNote(noteId, formData);
    } else {
      await notesApi.createNote(formData);
    }

    // Reset form or redirect
  };

  return (
    <form onSubmit={handleSubmit}>
      <input
        type="text"
        placeholder="Title"
        value={formData.title}
        onChange={(e) => setFormData(prev => ({ ...prev, title: e.target.value }))}
      />

      <textarea
        placeholder="Content"
        value={formData.content}
        onChange={(e) => setFormData(prev => ({ ...prev, content: e.target.value }))}
      />

      <TagInput
        tags={formData.tags}
        onChange={(tags) => setFormData(prev => ({ ...prev, tags }))}
      />

      <button type="submit">
        {noteId ? 'Update' : 'Create'} Note
      </button>
    </form>
  );
};
```

---

## 🎯 Use Cases & Examples

### Use Case 1: Personal To-Do List

```typescript
const createTodoNote = async () => {
  await notesApi.createNote({
    title: "Shopping List",
    content: "- Milk\n- Eggs\n- Bread\n- Butter",
    tags: ["todo", "shopping"]
  });
};
```

### Use Case 2: Meeting Notes

```typescript
const createMeetingNote = async () => {
  await notesApi.createNote({
    title: "Q4 Planning Meeting",
    content: "Attendees: John, Mary, Bob\n\nAgenda:\n1. Review Q3\n2. Set Q4 goals\n3. Budget allocation",
    tags: ["work", "meeting", "q4", "planning"]
  });
};
```

### Use Case 3: Quick Memo

```typescript
const createQuickMemo = async () => {
  await notesApi.createNote({
    title: "",  // No title
    content: "Remember to call dentist tomorrow at 2pm",
    tags: ["reminder"]
  });
};
```

### Use Case 4: Tag-based Organization

```typescript
// ดูบันทึกที่เกี่ยวกับงาน
const workNotes = await notesApi.getNotesByTag("work");

// ดูบันทึกส่วนตัว
const personalNotes = await notesApi.getNotesByTag("personal");
```

---

## 🔍 Important Notes

### 1. Privacy & Security
- แต่ละ user เห็นเฉพาะบันทึกของตัวเองเท่านั้น
- ไม่สามารถ share หรือ collaborate บันทึกกับคนอื่นได้
- ต้อง authenticate ทุก request

### 2. Tags Management
- Tags เก็บเป็น array of strings
- Case-sensitive (แนะนำให้ lowercase ทั้งหมด)
- ไม่จำกัดจำนวน tags
- ตอน update จะ replace tags เดิมทั้งหมด

### 3. Search Functionality
- ค้นหาทั้ง title และ content
- ไม่ค้นหา tags (ใช้ `/by-tag` แทน)
- รองรับ full-text search

### 4. Pagination
- Default limit: 20
- Max limit: 100
- ใช้ offset-based pagination

### 5. Pin Feature
- ไม่จำกัดจำนวนบันทึกที่ปักหมุดได้
- บันทึกที่ปักหมุดจะแสดงด้านบนใน UI (ขึ้นกับ frontend implementation)

---

## 🧪 Testing Checklist

### Basic CRUD
- [ ] สร้างบันทึกใหม่
- [ ] ดูรายการบันทึก
- [ ] ดูบันทึกเฉพาะ
- [ ] แก้ไขบันทึก
- [ ] ลบบันทึก

### Pin Feature
- [ ] ปักหมุดบันทึก
- [ ] ยกเลิกปักหมุด
- [ ] ดูรายการบันทึกที่ปักหมุด

### Search & Filter
- [ ] ค้นหาบันทึก (title)
- [ ] ค้นหาบันทึก (content)
- [ ] กรองตาม tag
- [ ] ทดสอบ pagination

### Edge Cases
- [ ] สร้างบันทึกโดยไม่มี title
- [ ] สร้างบันทึกโดยไม่มี content
- [ ] สร้างบันทึกโดยไม่มี tags
- [ ] ปักหมุดบันทึกที่ปักหมุดแล้ว (ควรได้ error)
- [ ] ยกเลิกปักหมุดบันทึกที่ไม่ได้ปักหมุด (ควรได้ error)

---

## 📞 Support & Questions

หากมีคำถามหรือพบปัญหาในการใช้งาน API นี้:
- ตรวจสอบ error response เพื่อดู error message
- ตรวจสอบว่าได้ส่ง Authorization header ครบถ้วน
- ตรวจสอบว่า UUIDs ถูกต้อง (format: `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`)

---

**Documentation Version:** 1.0
**API Version:** v1
**Last Tested:** 2025-12-01
