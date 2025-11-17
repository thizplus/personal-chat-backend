# API สำหรับเปลี่ยนชื่อและภาพ Group Chat

**วันที่:** 2025-11-17
**Version:** 1.0

---

## 📍 API Endpoint

### PATCH `/conversations/:conversationId`

**Description:** อัพเดตข้อมูล conversation (ชื่อ, ภาพ)

**Location:** `interfaces/api/handler/conversation_handler.go:593-688`

---

## 🔐 Authentication

**Required:** ✅ Yes

**Header:**
```
Authorization: Bearer <access_token>
```

**Permission:**
- User ต้องเป็น **member** ของ conversation
- (ถ้าต้องการจำกัดว่าแค่ admin → ต้องเพิ่ม permission check)

---

## 📥 Request

### URL Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `conversationId` | UUID | ✅ Yes | ID ของ conversation ที่ต้องการแก้ไข |

### Request Body

**Content-Type:** `application/json`

```json
{
  "title": "ชื่อกลุ่มใหม่",
  "icon_url": "https://example.com/new-icon.jpg"
}
```

### Body Parameters

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `title` | string | ⚪ Optional | ชื่อใหม่ของ group chat |
| `icon_url` | string | ⚪ Optional | URL ของภาพ icon ใหม่ |

**หมายเหตุ:**
- ส่งได้ทั้ง 2 field พร้อมกัน หรือส่งเพียง field เดียวก็ได้
- ถ้าส่งมาเป็น empty string (`""`) จะไม่ update field นั้น
- ต้องมีอย่างน้อย 1 field ที่ไม่ว่าง (ไม่งั้นจะ error)

---

## 📤 Response

### Success Response (200 OK)

```json
{
  "success": true,
  "message": "Conversation updated successfully"
}
```

### Error Responses

#### 400 Bad Request - Invalid Conversation ID
```json
{
  "success": false,
  "message": "Invalid conversation ID"
}
```

#### 400 Bad Request - No Changes
```json
{
  "success": false,
  "message": "No changes to update"
}
```

#### 401 Unauthorized
```json
{
  "success": false,
  "message": "Unauthorized: <error details>"
}
```

#### 403 Forbidden - Not a Member
```json
{
  "success": false,
  "message": "You are not a member of this conversation"
}
```

#### 500 Internal Server Error
```json
{
  "success": false,
  "message": "Failed to update conversation: <error details>"
}
```

---

## 🔄 WebSocket Event

เมื่อ conversation ถูก update สำเร็จ Backend จะส่ง WebSocket event ไปยัง **สมาชิกทุกคน** ใน conversation

### Event Type

```
conversation.update
```

### Event Location

**Service:** `application/serviceimpl/notification_service.go:302-305`
```go
func (s *notificationService) NotifyConversationUpdated(conversationID uuid.UUID, update interface{}) {
    s.wsPort.BroadcastConversationUpdated(conversationID, update)
}
```

**Adapter:** `infrastructure/adapter/websocket_adapter.go:92-95`
```go
func (a *WebSocketAdapter) BroadcastConversationUpdated(conversationID uuid.UUID, update interface{}) {
    a.BroadcastToConversation(conversationID, "conversation.update", update)
}
```

### Event Payload

**ตัวอย่าง Event ที่ส่งไป:**

#### เปลี่ยนทั้งชื่อและภาพ
```json
{
  "type": "conversation.update",
  "data": {
    "conversation_id": "123e4567-e89b-12d3-a456-426614174000",
    "title": "ชื่อกลุ่มใหม่",
    "icon_url": "https://example.com/new-icon.jpg"
  },
  "timestamp": "2025-11-17T10:30:00Z",
  "success": true
}
```

#### เปลี่ยนเฉพาะชื่อ
```json
{
  "type": "conversation.update",
  "data": {
    "conversation_id": "123e4567-e89b-12d3-a456-426614174000",
    "title": "ชื่อกลุ่มใหม่"
  },
  "timestamp": "2025-11-17T10:30:00Z",
  "success": true
}
```

#### เปลี่ยนเฉพาะภาพ
```json
{
  "type": "conversation.update",
  "data": {
    "conversation_id": "123e4567-e89b-12d3-a456-426614174000",
    "icon_url": "https://example.com/new-icon.jpg"
  },
  "timestamp": "2025-11-17T10:30:00Z",
  "success": true
}
```

### Broadcast Target

**ส่งไปที่:** สมาชิกทุกคนที่ **subscribe** ใน conversation นั้น

**วิธีการส่ง:** `BroadcastToConversation(conversationID, ...)`
- ส่งไปยังทุก client ที่อยู่ใน `h.conversationSubs[conversationID]`
- รวมทั้งคนที่ update ด้วย (ไม่มี excludeID)

---

## 💻 ตัวอย่างการใช้งาน

### cURL Example

#### Update ทั้งชื่อและภาพ
```bash
curl -X PATCH https://api.example.com/conversations/123e4567-e89b-12d3-a456-426614174000 \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "กลุ่มเพื่อนๆ 2024",
    "icon_url": "https://example.com/group-icons/friends-2024.jpg"
  }'
```

#### Update เฉพาะชื่อ
```bash
curl -X PATCH https://api.example.com/conversations/123e4567-e89b-12d3-a456-426614174000 \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "กลุ่มเพื่อนๆ 2024"
  }'
```

#### Update เฉพาะภาพ
```bash
curl -X PATCH https://api.example.com/conversations/123e4567-e89b-12d3-a456-426614174000 \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "icon_url": "https://example.com/group-icons/new-icon.jpg"
  }'
```

---

### JavaScript/TypeScript Example

#### Using Fetch API

```typescript
const updateGroupChat = async (
  conversationId: string,
  updates: { title?: string; icon_url?: string }
) => {
  const response = await fetch(`/conversations/${conversationId}`, {
    method: 'PATCH',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(updates),
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.message);
  }

  return await response.json();
};

// ใช้งาน
try {
  await updateGroupChat('123e4567-e89b-12d3-a456-426614174000', {
    title: 'กลุ่มเพื่อนๆ 2024',
    icon_url: 'https://example.com/new-icon.jpg',
  });
  console.log('Updated successfully!');
} catch (error) {
  console.error('Failed to update:', error);
}
```

#### WebSocket Event Listener

```typescript
// ฟัง WebSocket event
socket.on('conversation.update', (data) => {
  console.log('Conversation updated:', data);

  const { conversation_id, title, icon_url } = data;

  // Update UI
  updateConversationInStore({
    id: conversation_id,
    ...(title && { title }),
    ...(icon_url && { icon_url }),
  });
});
```

---

### React Example (with Zustand)

```typescript
// API Service
export const conversationApi = {
  updateGroupChat: async (
    conversationId: string,
    updates: { title?: string; icon_url?: string }
  ) => {
    const response = await apiClient.patch(
      `/conversations/${conversationId}`,
      updates
    );
    return response.data;
  },
};

// Store
interface ConversationStore {
  conversations: Conversation[];
  updateConversation: (id: string, updates: Partial<Conversation>) => void;
}

export const useConversationStore = create<ConversationStore>((set) => ({
  conversations: [],

  updateConversation: (id, updates) =>
    set((state) => ({
      conversations: state.conversations.map((conv) =>
        conv.id === id ? { ...conv, ...updates } : conv
      ),
    })),
}));

// Component
const UpdateGroupChatModal = ({ conversationId, onClose }) => {
  const [title, setTitle] = useState('');
  const [iconUrl, setIconUrl] = useState('');
  const [loading, setLoading] = useState(false);

  const handleUpdate = async () => {
    setLoading(true);
    try {
      await conversationApi.updateGroupChat(conversationId, {
        title,
        icon_url: iconUrl,
      });

      toast.success('อัพเดตสำเร็จ!');
      onClose();
    } catch (error) {
      toast.error('อัพเดตไม่สำเร็จ: ' + error.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal>
      <Input
        value={title}
        onChange={(e) => setTitle(e.target.value)}
        placeholder="ชื่อกลุ่มใหม่"
      />
      <Input
        value={iconUrl}
        onChange={(e) => setIconUrl(e.target.value)}
        placeholder="URL ภาพโปรไฟล์"
      />
      <Button onClick={handleUpdate} disabled={loading}>
        {loading ? 'กำลังบันทึก...' : 'บันทึก'}
      </Button>
    </Modal>
  );
};

// WebSocket Hook
useEffect(() => {
  if (!socket) return;

  const handleConversationUpdate = (data: any) => {
    console.log('[WS] conversation.update:', data);

    const { conversation_id, title, icon_url } = data;

    useConversationStore.getState().updateConversation(conversation_id, {
      ...(title && { title }),
      ...(icon_url && { iconUrl: icon_url }),
    });
  };

  socket.on('conversation.update', handleConversationUpdate);

  return () => {
    socket.off('conversation.update', handleConversationUpdate);
  };
}, [socket]);
```

---

## 🔍 Backend Code Flow

### 1. API Handler
**File:** `conversation_handler.go:593-688`

```go
func (h *ConversationHandler) UpdateConversation(c *fiber.Ctx) error {
    // 1. ตรวจสอบ auth
    userID := middleware.GetUserUUID(c)

    // 2. ตรวจสอบว่าเป็น member
    isMember := h.conversationService.CheckMembership(userID, conversationID)

    // 3. Parse body
    var input struct {
        Title   string `json:"title"`
        IconURL string `json:"icon_url"`
    }

    // 4. Build update data
    updateData := types.JSONB{}
    if input.Title != "" {
        updateData["title"] = input.Title
    }
    if input.IconURL != "" {
        updateData["icon_url"] = input.IconURL
    }

    // 5. Update conversation
    err = h.conversationService.UpdateConversation(conversationID, updateData)

    // 6. Send WebSocket notification
    h.notificationService.NotifyConversationUpdated(conversationID, updateData)

    return c.JSON(...)
}
```

### 2. Service Layer
**File:** `conversations_service.go:944-947`

```go
func (s *conversationService) UpdateConversation(id uuid.UUID, updateData types.JSONB) error {
    return s.conversationRepo.UpdateConversation(id, updateData)
}
```

### 3. Repository Layer
**File:** `conversation_repository.go`

```go
func (r *conversationRepository) UpdateConversation(id uuid.UUID, updateData types.JSONB) error {
    return r.db.Model(&models.Conversation{}).
        Where("id = ?", id).
        Updates(updateData).Error
}
```

### 4. WebSocket Notification
**File:** `notification_service.go:302-305`

```go
func (s *notificationService) NotifyConversationUpdated(conversationID uuid.UUID, update interface{}) {
    s.wsPort.BroadcastConversationUpdated(conversationID, update)
}
```

**File:** `websocket_adapter.go:92-95`

```go
func (a *WebSocketAdapter) BroadcastConversationUpdated(conversationID uuid.UUID, update interface{}) {
    a.BroadcastToConversation(conversationID, "conversation.update", update)
}
```

---

## 🎯 ข้อควรระวัง

### 1. **Permission Control** ⚠️

**ปัจจุบัน:** ทุกคนที่เป็น member สามารถแก้ไขชื่อและภาพได้

**แนะนำ:** ควรเพิ่ม permission check ว่าต้องเป็น **admin** เท่านั้น

**วิธีแก้:**
```go
// เพิ่มใน UpdateConversation handler
member, err := h.conversationRepo.GetMember(conversationID, userID)
if !member.IsAdmin {
    return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
        "success": false,
        "message": "Only admins can update conversation details",
    })
}
```

### 2. **Empty String Handling**

ปัจจุบัน: ถ้าส่ง `""` (empty string) มาจะ**ไม่**อัพเดต field นั้น

ถ้าต้องการ **ลบค่า** (set เป็น null/empty):
```go
// ใช้ pointer แทน
type UpdateInput struct {
    Title   *string `json:"title"`
    IconURL *string `json:"icon_url"`
}

// ตรวจสอบ nil vs empty
if input.Title != nil {
    updateData["title"] = *input.Title
}
```

### 3. **Image Upload**

ปัจจุบัน API รับเฉพาะ **URL** ของภาพ

ถ้าต้องการให้ user **upload ภาพ** ต้องสร้าง API แยกสำหรับ upload:
```
POST /conversations/:conversationId/icon
Content-Type: multipart/form-data
```

แล้วค่อยเอา URL ที่ได้มา update

### 4. **File Validation**

ควรเพิ่มการ validate:
- ✅ URL format ถูกต้อง
- ✅ ไฟล์เป็น image type (jpg, png, gif)
- ✅ ขนาดไฟล์ไม่เกิน limit

---

## 📋 Checklist สำหรับ Frontend Developer

เมื่อทำ feature นี้ ควรตรวจสอบ:

- [ ] ✅ เรียก API ด้วย PATCH method (ไม่ใช่ PUT)
- [ ] ✅ ส่ง Authorization header
- [ ] ✅ Handle loading state
- [ ] ✅ Handle error cases (400, 403, 500)
- [ ] ✅ ฟัง WebSocket event `conversation.update`
- [ ] ✅ Update conversation ใน store เมื่อได้รับ event
- [ ] ✅ แสดง notification เมื่อสำเร็จ
- [ ] ✅ Validate input (title ไม่ว่าง, URL format ถูกต้อง)
- [ ] ✅ แสดง UI สำหรับ upload image (ถ้ามี)
- [ ] ✅ Optimistic update (update UI ทันที แล้วค่อย rollback ถ้า fail)

---

## 🚀 Features ที่ควรเพิ่ม (แนะนำ)

### 1. Admin-Only Permission
ควรจำกัดให้แค่ admin แก้ได้

### 2. Image Upload API
```
POST /conversations/:conversationId/icon
```

### 3. History Tracking
เก็บประวัติการเปลี่ยนชื่อ/ภาพ

### 4. System Message
สร้างข้อความระบบเมื่อมีการเปลี่ยน:
```
"User A เปลี่ยนชื่อกลุ่มเป็น 'ชื่อใหม่'"
"User B เปลี่ยนรูปโปรไฟล์กลุ่ม"
```

### 5. Validation
- Title: ความยาว 1-100 ตัวอักษร
- Icon URL: ต้องเป็น https:// และเป็นรูปภาพ
- Rate limiting: ไม่ให้เปลี่ยนบ่อยเกินไป

---

**เอกสารนี้สร้างขึ้นเมื่อ:** 2025-11-17
**Version:** 1.0
**Status:** ✅ Production Ready (แต่แนะนำเพิ่ม admin-only permission)
