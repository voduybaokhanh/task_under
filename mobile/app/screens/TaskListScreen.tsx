import React, { useEffect, useState, useMemo } from 'react';
import {
  View,
  Text,
  FlatList,
  TouchableOpacity,
  StyleSheet,
  ActivityIndicator,
  RefreshControl,
  TextInput,
} from 'react-native';
import { useNavigation } from '@react-navigation/native';
import { useTaskStore } from '../../store/useTaskStore';
import { apiService } from '../../services/api';
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

function timeRemaining(deadline: string): string {
  const diff = new Date(deadline).getTime() - Date.now();
  if (diff <= 0) return 'Expired';
  const h = Math.floor(diff / 3600000);
  if (h < 24) return `${h}h left`;
  const d = Math.floor(h / 24);
  return `${d}d left`;
}

const FILTERS = ['all', 'open', 'claimed', 'completed'] as const;
type Filter = typeof FILTERS[number];

export default function TaskListScreen() {
  const navigation = useNavigation<any>();
  const { tasks, loading, error, fetchOpenTasks } = useTaskStore();
  const [search, setSearch] = useState('');
  const [filter, setFilter] = useState<Filter>('all');
  // Server-side full-text search results; null means "not searching" → use store list.
  const [searchResults, setSearchResults] = useState<Task[] | null>(null);
  const [searching, setSearching] = useState(false);

  useEffect(() => {
    fetchOpenTasks();
  }, []);

  // Debounced full-text search: hits GET /tasks/search 300ms after the last keystroke.
  useEffect(() => {
    const q = search.trim();
    if (!q) {
      setSearchResults(null);
      setSearching(false);
      return;
    }

    setSearching(true);
    const handle = setTimeout(async () => {
      try {
        const status = filter === 'all' ? undefined : filter;
        const results = await apiService.searchTasks(q, status);
        setSearchResults(results);
      } catch {
        setSearchResults([]);
      } finally {
        setSearching(false);
      }
    }, 300);

    return () => clearTimeout(handle);
  }, [search, filter]);

  const filtered = useMemo(() => {
    if (searchResults !== null) return searchResults;
    return tasks.filter((t) => filter === 'all' || t.status === filter);
  }, [tasks, filter, searchResults]);

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
      <Text style={styles.taskDescription} numberOfLines={2}>{item.description}</Text>
      <View style={styles.cardBottom}>
        <Text style={styles.metaText}>{timeRemaining(item.claim_deadline)}</Text>
        <Text style={styles.metaText}>{item.max_claimants} slot{item.max_claimants !== 1 ? 's' : ''}</Text>
      </View>
    </TouchableOpacity>
  );

  return (
    <View style={styles.container}>
      <View style={styles.header}>
        <Text style={styles.headerTitle}>Tasks</Text>
        <TouchableOpacity
          style={styles.createButton}
          onPress={() => navigation.navigate('CreateTask' as never)}
        >
          <Text style={styles.createButtonText}>+ New</Text>
        </TouchableOpacity>
      </View>

      <View style={styles.searchRow}>
        <TextInput
          style={styles.searchInput}
          value={search}
          onChangeText={setSearch}
          placeholder="Search tasks..."
          placeholderTextColor="#555"
        />
        {searching && (
          <ActivityIndicator size="small" color="#4CAF50" style={styles.searchSpinner} />
        )}
      </View>

      <View style={styles.filterRow}>
        {FILTERS.map((f) => (
          <TouchableOpacity
            key={f}
            style={[styles.filterChip, filter === f && styles.filterChipActive]}
            onPress={() => setFilter(f)}
          >
            <Text style={[styles.filterText, filter === f && styles.filterTextActive]}>
              {f.charAt(0).toUpperCase() + f.slice(1)}
            </Text>
          </TouchableOpacity>
        ))}
      </View>

      {error && <Text style={styles.error}>{error}</Text>}

      {loading && tasks.length === 0 ? (
        <ActivityIndicator size="large" color="#4CAF50" style={styles.loader} />
      ) : (
        <FlatList
          data={filtered}
          renderItem={renderTask}
          keyExtractor={(item) => item.id}
          refreshControl={<RefreshControl refreshing={loading} onRefresh={fetchOpenTasks} tintColor="#4CAF50" />}
          contentContainerStyle={styles.listContent}
          ListEmptyComponent={
            <View style={styles.emptyContainer}>
              <Text style={styles.emptyText}>
                {search ? 'No matching tasks' : 'No tasks available'}
              </Text>
            </View>
          }
        />
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: '#000' },
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingHorizontal: 16,
    paddingVertical: 12,
    backgroundColor: '#111',
  },
  headerTitle: { fontSize: 22, fontWeight: 'bold', color: '#fff' },
  createButton: {
    backgroundColor: '#4CAF50',
    paddingHorizontal: 16,
    paddingVertical: 8,
    borderRadius: 8,
  },
  createButtonText: { color: '#fff', fontWeight: '700', fontSize: 14 },
  searchRow: { padding: 12, paddingBottom: 4, justifyContent: 'center' },
  searchSpinner: { position: 'absolute', right: 24, top: 22 },
  searchInput: {
    backgroundColor: '#111',
    borderWidth: 1,
    borderColor: '#333',
    borderRadius: 10,
    paddingHorizontal: 14,
    paddingVertical: 10,
    color: '#fff',
    fontSize: 15,
  },
  filterRow: {
    flexDirection: 'row',
    paddingHorizontal: 12,
    paddingVertical: 8,
    gap: 8,
  },
  filterChip: {
    paddingHorizontal: 12,
    paddingVertical: 6,
    borderRadius: 16,
    borderWidth: 1,
    borderColor: '#333',
    backgroundColor: '#111',
  },
  filterChipActive: {
    backgroundColor: '#4CAF50',
    borderColor: '#4CAF50',
  },
  filterText: { color: '#777', fontSize: 13, fontWeight: '500' },
  filterTextActive: { color: '#fff' },
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
  badge: {
    paddingHorizontal: 8,
    paddingVertical: 3,
    borderRadius: 6,
  },
  badgeText: { fontSize: 10, fontWeight: '700', color: '#fff', letterSpacing: 0.5 },
  taskReward: { fontSize: 22, fontWeight: 'bold', color: '#4CAF50', marginBottom: 6 },
  taskDescription: { fontSize: 13, color: '#777', lineHeight: 18, marginBottom: 10 },
  cardBottom: { flexDirection: 'row', justifyContent: 'space-between' },
  metaText: { fontSize: 12, color: '#555' },
  emptyContainer: { alignItems: 'center', marginTop: 60 },
  emptyText: { color: '#555', fontSize: 16 },
});
