import React, { useEffect } from 'react';
import {
  View,
  Text,
  FlatList,
  TouchableOpacity,
  StyleSheet,
  ActivityIndicator,
  RefreshControl,
} from 'react-native';
import { useNavigation } from '@react-navigation/native';
import { useTaskStore } from '../../store/useTaskStore';
import { Task } from '../../types';

const STATUS_COLORS: Record<string, string> = {
  open: '#4CAF50',
  claimed: '#FF9800',
  completed: '#2196F3',
  cancelled: '#555',
  disputed: '#F44336',
};

function StatusBadge({ status }: { status: string }) {
  return (
    <View style={[styles.badge, { backgroundColor: STATUS_COLORS[status] ?? '#555' }]}>
      <Text style={styles.badgeText}>{status.toUpperCase()}</Text>
    </View>
  );
}

export default function MyTasksScreen() {
  const navigation = useNavigation<any>();
  const { myTasks, loading, error, fetchMyTasks } = useTaskStore();

  useEffect(() => {
    fetchMyTasks();
  }, []);

  const renderTask = ({ item }: { item: Task }) => (
    <TouchableOpacity
      style={styles.taskCard}
      onPress={() => navigation.navigate('TaskDetail' as never, { taskId: item.id } as never)}
      activeOpacity={0.75}
    >
      <View style={styles.cardTop}>
        <Text style={styles.taskTitle} numberOfLines={1}>{item.title}</Text>
        <StatusBadge status={item.status} />
      </View>
      <Text style={styles.taskReward}>${item.reward_amount.toFixed(2)}</Text>
      <Text style={styles.taskMeta}>
        Created {new Date(item.created_at).toLocaleDateString()}
      </Text>
    </TouchableOpacity>
  );

  const open = myTasks.filter((t) => t.status === 'open').length;
  const completed = myTasks.filter((t) => t.status === 'completed').length;

  return (
    <View style={styles.container}>
      <View style={styles.headerBox}>
        <Text style={styles.headerTitle}>My Tasks</Text>
        <View style={styles.summaryRow}>
          <Text style={styles.summaryItem}>{open} open</Text>
          <Text style={styles.summaryDot}>·</Text>
          <Text style={styles.summaryItem}>{completed} completed</Text>
        </View>
      </View>

      {error && <Text style={styles.error}>{error}</Text>}

      {loading && myTasks.length === 0 ? (
        <ActivityIndicator size="large" color="#4CAF50" style={styles.loader} />
      ) : (
        <FlatList
          data={myTasks}
          renderItem={renderTask}
          keyExtractor={(item) => item.id}
          refreshControl={<RefreshControl refreshing={loading} onRefresh={fetchMyTasks} tintColor="#4CAF50" />}
          contentContainerStyle={styles.listContent}
          ListEmptyComponent={
            <View style={styles.emptyContainer}>
              <Text style={styles.emptyText}>You haven't created any tasks yet</Text>
            </View>
          }
        />
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: '#000' },
  headerBox: {
    padding: 16,
    backgroundColor: '#111',
    borderBottomWidth: 1,
    borderBottomColor: '#222',
  },
  headerTitle: { fontSize: 22, fontWeight: 'bold', color: '#fff', marginBottom: 4 },
  summaryRow: { flexDirection: 'row', alignItems: 'center', gap: 6 },
  summaryItem: { fontSize: 13, color: '#666' },
  summaryDot: { color: '#444' },
  error: { color: '#ff4444', padding: 16, textAlign: 'center' },
  loader: { marginTop: 50 },
  listContent: { padding: 12 },
  taskCard: {
    backgroundColor: '#111',
    padding: 16,
    marginBottom: 10,
    borderRadius: 12,
    borderWidth: 1,
    borderColor: '#222',
  },
  cardTop: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 6,
  },
  taskTitle: { fontSize: 16, fontWeight: 'bold', color: '#fff', flex: 1, marginRight: 8 },
  badge: { paddingHorizontal: 8, paddingVertical: 3, borderRadius: 6 },
  badgeText: { fontSize: 10, fontWeight: '700', color: '#fff', letterSpacing: 0.5 },
  taskReward: { fontSize: 20, fontWeight: 'bold', color: '#4CAF50', marginBottom: 6 },
  taskMeta: { fontSize: 12, color: '#555' },
  emptyContainer: { alignItems: 'center', marginTop: 60 },
  emptyText: { color: '#555', fontSize: 16 },
});
