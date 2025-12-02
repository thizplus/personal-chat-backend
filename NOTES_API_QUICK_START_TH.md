# Notes API - Quick Start (ภาษาไทย)

**วันที่:** 2025-12-03
**สถานะ:** ✅ **พร้อมใช้งาน**

---

## 🎯 สรุปสั้น ๆ

### Features ที่มี
- ✅ **Personal Notes** - บันทึกส่วนตัว
- ✅ **Conversation Notes** - บันทึกเฉพาะการสนทนา
- ✅ **Pin/Unpin** - ปักหมุดบันทึก
- ✅ **Search** - ค้นหาบันทึก
- ✅ **Tags** - จัดกลุ่มด้วย tag

### ⚠️ สิ่งสำคัญที่ต้องรู้
1. **Pin endpoint รองรับทั้ง PUT และ POST**
2. **Notes เป็นของ user เท่านั้น** (ไม่แชร์ให้คนอื่น)
3. **Conversation notes ต้องเป็น member** (backend เช็คให้อัตโนมัติ)

---

## 📋 API Endpoints (สรุป)

| Method | URL | ทำอะไร |
|--------|-----|--------|
| `POST` | `/notes` | สร้างบันทึกใหม่ |
| `GET` | `/notes` | ดึงบันทึกทั้งหมด |
| `GET` | `/notes?conversation_id=xxx` | ดึงบันทึกของ conversation |
| `GET` | `/notes?scope=global` | ดึงเฉพาะบันทึกส่วนตัว |
| `PUT/POST` | `/notes/:id/pin` | ปักหมุด (ใช้ PUT หรือ POST ก็ได้) |
| `DELETE` | `/notes/:id/pin` | ยกเลิกปักหมุด |
| `GET` | `/notes/pinned` | ดึงบันทึกที่ปักหมุด |
| `GET` | `/notes/search?q=xxx` | ค้นหา |
| `PUT` | `/notes/:id` | อัปเดต |
| `DELETE` | `/notes/:id` | ลบ |

---

## 💻 ตัวอย่างการใช้งาน

### 1. สร้างบันทึกส่วนตัว (Global Note)

```typescript
// ไม่ต้องส่ง conversation_id
await fetch('/api/v1/notes', {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    title: "Shopping List",
    content: "นม ไข่ ขนมปัง",
    tags: ["personal"]
  })
});
```

### 2. สร้างบันทึกสำหรับ Conversation

```typescript
// ส่ง conversation_id ไปด้วย
await fetch('/api/v1/notes', {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    conversation_id: "abc-123-def",  // 🆕 ส่งไปด้วย
    title: "Meeting Notes",
    content: "บันทึกการประชุม...",
    tags: ["work", "meeting"]
  })
});
```

### 3. ดึงบันทึกทั้งหมด

```typescript
// ดึงทั้ง global + conversation notes
const response = await fetch('/api/v1/notes', {
  headers: { 'Authorization': `Bearer ${token}` }
});
const data = await response.json();
console.log(data.data.notes);
```

### 4. ดึงเฉพาะบันทึกส่วนตัว

```typescript
// เฉพาะ global notes
const response = await fetch('/api/v1/notes?scope=global', {
  headers: { 'Authorization': `Bearer ${token}` }
});
```

### 5. ดึงบันทึกของ Conversation

```typescript
// เฉพาะ conversation นั้น ๆ
const conversationId = "abc-123-def";
const response = await fetch(`/api/v1/notes?conversation_id=${conversationId}`, {
  headers: { 'Authorization': `Bearer ${token}` }
});
```

### 6. ปักหมุดบันทึก

```typescript
const noteId = "note-uuid";

// ใช้ POST (แนะนำ)
await fetch(`/api/v1/notes/${noteId}/pin`, {
  method: 'POST',  // หรือ PUT ก็ได้
  headers: { 'Authorization': `Bearer ${token}` }
});
```

### 7. ยกเลิกปักหมุด

```typescript
await fetch(`/api/v1/notes/${noteId}/pin`, {
  method: 'DELETE',
  headers: { 'Authorization': `Bearer ${token}` }
});
```

### 8. ค้นหาบันทึก

```typescript
const query = "meeting";
const response = await fetch(`/api/v1/notes/search?q=${query}`, {
  headers: { 'Authorization': `Bearer ${token}` }
});
```

---

## 📦 Response Format

### สำเร็จ (Create/Update)
```json
{
  "success": true,
  "message": "Note created successfully",
  "data": {
    "id": "note-uuid",
    "user_id": "user-uuid",
    "conversation_id": "conv-uuid",  // หรือ null
    "title": "Title",
    "content": "Content",
    "tags": ["tag1", "tag2"],
    "is_pinned": false,
    "created_at": "2025-12-03T10:00:00Z",
    "updated_at": "2025-12-03T10:00:00Z"
  }
}
```

### สำเร็จ (List)
```json
{
  "success": true,
  "data": {
    "notes": [
      {
        "id": "note-1",
        "conversation_id": null,  // Global note
        "title": "Shopping List",
        ...
      },
      {
        "id": "note-2",
        "conversation_id": "conv-abc",  // Conversation note
        "title": "Meeting Notes",
        ...
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

### Error
```json
{
  "success": false,
  "message": "error message here"
}
```

---

## 🔐 Permissions & Privacy

### Personal Notes (Global)
- เป็นของ user เท่านั้น
- ไม่ต้องเช็ค permission

### Conversation Notes
- ต้องเป็น member ของ conversation
- Backend เช็ค permission อัตโนมัติ
- ถ้าไม่ใช่ member จะได้ `403 Forbidden`

### Privacy
- **Notes เป็นส่วนตัว** ของ user ที่สร้าง
- **ไม่แชร์** ให้ member คนอื่นในกลุ่ม
- เหมือน "บันทึกส่วนตัวเกี่ยวกับ conversation นั้น"

---

## 🎨 UI Suggestions

### Global Notes Page
```
┌─────────────────────────────────┐
│  My Personal Notes              │
├─────────────────────────────────┤
│  [Search...] [+ Create]         │
│                                 │
│  📌 Shopping List               │
│     Milk, Eggs, Bread           │
│                                 │
│  📝 Work Todo                   │
│     - Finish report             │
│     - Call client               │
└─────────────────────────────────┘
```

### Conversation Sidebar
```
┌─────────────────────────────────┐
│  Conversation: Project Alpha    │
├─────────────────────────────────┤
│  Members | Files | Notes ←      │
├─────────────────────────────────┤
│  📝 Notes (3)                   │
│                                 │
│  📌 Meeting 12/01               │
│     Deadline: Friday            │
│                                 │
│  📝 Action Items                │
│     - Review design             │
│                                 │
│  [+ Add Note]                   │
└─────────────────────────────────┘
```

---

## ⚠️ Common Errors & Solutions

### 1. "Method Not Allowed"
**ปัญหา:** ใช้ HTTP method ผิด

**แก้:**
```typescript
// ❌ Wrong
fetch('/notes/xxx/pin', { method: 'GET' })

// ✅ Correct
fetch('/notes/xxx/pin', { method: 'POST' })  // or PUT
```

### 2. "Invalid conversation_id format"
**ปัญหา:** conversation_id ไม่ใช่ UUID

**แก้:**
```typescript
// ❌ Wrong
conversation_id: "123"

// ✅ Correct
conversation_id: "550e8400-e29b-41d4-a716-446655440000"
```

### 3. "user is not a member of this conversation"
**ปัญหา:** User ไม่ได้เป็น member ของ conversation

**แก้:**
- เช็คว่า user เป็น member หรือยัง
- หรือสร้างเป็น global note แทน (ไม่ส่ง conversation_id)

---

## 📚 TypeScript Types

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
```

---

## 🧪 ทดสอบด้วย cURL

### สร้าง Global Note
```bash
curl -X POST https://b01.ngrok.dev/api/v1/notes \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Test Note",
    "content": "Testing...",
    "tags": ["test"]
  }'
```

### สร้าง Conversation Note
```bash
curl -X POST https://b01.ngrok.dev/api/v1/notes \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "conversation_id": "your-conversation-uuid",
    "title": "Meeting Notes",
    "content": "...",
    "tags": ["work"]
  }'
```

### ปักหมุด
```bash
curl -X POST https://b01.ngrok.dev/api/v1/notes/NOTE_ID/pin \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### ดึงบันทึก
```bash
# ทั้งหมด
curl https://b01.ngrok.dev/api/v1/notes \
  -H "Authorization: Bearer YOUR_TOKEN"

# เฉพาะ global
curl https://b01.ngrok.dev/api/v1/notes?scope=global \
  -H "Authorization: Bearer YOUR_TOKEN"

# เฉพาะ conversation
curl https://b01.ngrok.dev/api/v1/notes?conversation_id=CONV_ID \
  -H "Authorization: Bearer YOUR_TOKEN"
```

---

## ✅ Checklist สำหรับ Frontend

### Setup
- [ ] สร้าง NotesAPI client
- [ ] เพิ่ม TypeScript types
- [ ] เพิ่ม error handling

### UI Components
- [ ] NoteCard component
- [ ] CreateNoteModal
- [ ] NotesList component
- [ ] SearchBar

### Pages/Views
- [ ] Global Notes page
- [ ] Conversation notes sidebar

### Features
- [ ] Create global note
- [ ] Create conversation note
- [ ] Pin/Unpin note
- [ ] Search notes
- [ ] Filter by tags
- [ ] Delete note
- [ ] Update note

### Testing
- [ ] ทดสอบสร้าง global note
- [ ] ทดสอบสร้าง conversation note
- [ ] ทดสอบ pin/unpin
- [ ] ทดสอบ search
- [ ] ทดสอบ filter
- [ ] ทดสอบ permission (403 error)

---

## 🚀 Ready to Start!

**Backend:** ✅ พร้อมแล้ว
**Documentation:** ✅ ครบถ้วน
**API:** ✅ ใช้งานได้

**เอกสารเพิ่มเติม:**
1. `NOTES_API_COMPLETE_SUMMARY_FOR_FRONTEND.md` - Full API docs (อังกฤษ)
2. `NOTES_API_QUICK_START_TH.md` - ไฟล์นี้ (Quick start ภาษาไทย)

**มีปัญหาหรือข้อสงสัย?**
- ดูเอกสารฉบับเต็ม (ไฟล์ที่ 1)
- ทดสอบด้วย cURL ก่อน
- เช็ค error response

---

**สร้างเมื่อ:** 2025-12-03
**สถานะ:** ✅ Production Ready
**Backend URL:** https://b01.ngrok.dev/api/v1/notes
