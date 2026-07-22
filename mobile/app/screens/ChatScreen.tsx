import React, { useEffect, useRef, useState } from 'react';
import {
  View,
  Text,
  TextInput,
  TouchableOpacity,
  FlatList,
  StyleSheet,
  KeyboardAvoidingView,
  Platform,
  ActivityIndicator,
} from 'react-native';
import { useRoute, useNavigation } from '@react-navigation/native';
import { useChatStore } from '../../store/useChatStore';
import { Message } from '../../types';

export default function ChatScreen() {
  const route = useRoute();
  const navigation = useNavigation();
  const { taskId, claimerId } = route.params as { taskId: string; claimerId?: string };
  const { selectedChat, messages, loading, myUserId, encrypted, getOrCreateChat, sendMessage } =
    useChatStore();
  const [messageText, setMessageText] = useState('');
  const flatListRef = useRef<FlatList>(null);

  useEffect(() => {
    getOrCreateChat(taskId, claimerId);
  }, [taskId, claimerId]);

  useEffect(() => {
    navigation.setOptions({
      headerTitle: () => (
        <View>
          <Text style={styles.headerTitle}>Chat</Text>
          <Text style={encrypted ? styles.headerBadge : styles.headerBadgeOff}>
            {encrypted ? '🔒 E2E Encrypted' : '🔓 Not encrypted'}
          </Text>
        </View>
      ),
    });
  }, [navigation, encrypted]);

  const handleSend = async () => {
    const text = messageText.trim();
    if (!text || !selectedChat) return;
    setMessageText('');
    await sendMessage(selectedChat.id, text);
  };

  const chatMessages = selectedChat ? messages.get(selectedChat.id) || [] : [];

  const renderMessage = ({ item }: { item: Message }) => {
    const isMe = item.sender_id === myUserId;
    return (
      <View style={[styles.msgRow, isMe ? styles.msgRowMe : styles.msgRowOther]}>
        <View style={[styles.bubble, isMe ? styles.bubbleMe : styles.bubbleOther]}>
          <Text style={[styles.msgText, isMe ? styles.msgTextMe : styles.msgTextOther]}>
            {item.content}
          </Text>
          <Text style={[styles.msgTime, isMe ? styles.msgTimeMe : styles.msgTimeOther]}>
            {new Date(item.created_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
          </Text>
        </View>
      </View>
    );
  };

  if (loading && chatMessages.length === 0) {
    return (
      <View style={styles.center}>
        <ActivityIndicator size="large" color="#4CAF50" />
      </View>
    );
  }

  return (
    <KeyboardAvoidingView
      style={styles.container}
      behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
      keyboardVerticalOffset={90}
    >
      {chatMessages.length === 0 ? (
        <View style={styles.center}>
          <Text style={styles.emptyText}>No messages yet. Say hi!</Text>
        </View>
      ) : (
        <FlatList
          ref={flatListRef}
          data={[...chatMessages].reverse()}
          renderItem={renderMessage}
          keyExtractor={(item) => item.id}
          style={styles.list}
          contentContainerStyle={styles.listContent}
          inverted
        />
      )}

      <View style={styles.inputRow}>
        <TextInput
          style={styles.input}
          value={messageText}
          onChangeText={setMessageText}
          placeholder="Message..."
          placeholderTextColor="#555"
          multiline
          maxLength={2000}
          returnKeyType="send"
          onSubmitEditing={handleSend}
          blurOnSubmit={false}
        />
        <TouchableOpacity
          style={[styles.sendBtn, !messageText.trim() && styles.sendBtnDisabled]}
          onPress={handleSend}
          disabled={!messageText.trim()}
        >
          <Text style={styles.sendText}>Send</Text>
        </TouchableOpacity>
      </View>
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: '#000' },
  headerTitle: { color: '#fff', fontSize: 17, fontWeight: 'bold' },
  headerBadge: { color: '#4CAF50', fontSize: 11, marginTop: 1 },
  headerBadgeOff: { color: '#888', fontSize: 11, marginTop: 1 },
  center: { flex: 1, justifyContent: 'center', alignItems: 'center' },
  emptyText: { color: '#555', fontSize: 15 },
  list: { flex: 1 },
  listContent: { padding: 12 },
  msgRow: { marginBottom: 8, flexDirection: 'row' },
  msgRowMe: { justifyContent: 'flex-end' },
  msgRowOther: { justifyContent: 'flex-start' },
  bubble: {
    maxWidth: '78%',
    borderRadius: 18,
    paddingHorizontal: 14,
    paddingVertical: 10,
  },
  bubbleMe: {
    backgroundColor: '#4CAF50',
    borderBottomRightRadius: 4,
  },
  bubbleOther: {
    backgroundColor: '#1e1e1e',
    borderBottomLeftRadius: 4,
  },
  msgText: { fontSize: 15, lineHeight: 20 },
  msgTextMe: { color: '#fff' },
  msgTextOther: { color: '#eee' },
  msgTime: { fontSize: 11, marginTop: 4 },
  msgTimeMe: { color: 'rgba(255,255,255,0.6)', textAlign: 'right' },
  msgTimeOther: { color: '#555' },
  inputRow: {
    flexDirection: 'row',
    alignItems: 'flex-end',
    padding: 12,
    backgroundColor: '#111',
    borderTopWidth: 1,
    borderTopColor: '#222',
    gap: 10,
  },
  input: {
    flex: 1,
    backgroundColor: '#1e1e1e',
    borderRadius: 22,
    paddingHorizontal: 16,
    paddingVertical: 10,
    color: '#fff',
    fontSize: 15,
    maxHeight: 120,
  },
  sendBtn: {
    backgroundColor: '#4CAF50',
    paddingHorizontal: 18,
    paddingVertical: 10,
    borderRadius: 22,
    justifyContent: 'center',
  },
  sendBtnDisabled: { backgroundColor: '#333' },
  sendText: { color: '#fff', fontWeight: '700', fontSize: 14 },
});
