import React, { useEffect } from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  ActivityIndicator,
  TouchableOpacity,
  Alert,
} from 'react-native';
import AsyncStorage from '@react-native-async-storage/async-storage';
import { useUserStore } from '../../store/useUserStore';
import { useTaskStore } from '../../store/useTaskStore';

function StatCard({ label, value, color = '#fff' }: { label: string; value: string; color?: string }) {
  return (
    <View style={styles.statCard}>
      <Text style={[styles.statValue, { color }]}>{value}</Text>
      <Text style={styles.statLabel}>{label}</Text>
    </View>
  );
}

function ReputationBar({ score }: { score: number }) {
  const maxScore = 1000;
  const pct = Math.min(score / maxScore, 1);
  const color = score >= 500 ? '#4CAF50' : score >= 200 ? '#FF9800' : '#F44336';
  return (
    <View style={styles.repBarContainer}>
      <View style={styles.repBarBg}>
        <View style={[styles.repBarFill, { width: `${pct * 100}%` as any, backgroundColor: color }]} />
      </View>
      <Text style={[styles.repScore, { color }]}>{score} pts</Text>
    </View>
  );
}

export default function ProfileScreen() {
  const { me, loading, fetchMe } = useUserStore();
  const { myTasks, fetchMyTasks } = useTaskStore();

  useEffect(() => {
    fetchMe();
    fetchMyTasks();
  }, []);

  if (loading && !me) {
    return (
      <View style={styles.center}>
        <ActivityIndicator size="large" color="#4CAF50" />
      </View>
    );
  }

  const shortId = me ? me.id.slice(-8).toUpperCase() : '--------';
  const completedTasks = myTasks.filter((t) => t.status === 'completed').length;
  const openTasks = myTasks.filter((t) => t.status === 'open').length;

  return (
    <ScrollView style={styles.container}>
      <View style={styles.header}>
        <View style={styles.avatar}>
          <Text style={styles.avatarText}>?</Text>
        </View>
        <Text style={styles.userId}>#{shortId}</Text>
        <Text style={styles.anonBadge}>Anonymous</Text>
      </View>

      <View style={styles.section}>
        <Text style={styles.sectionTitle}>Reputation</Text>
        <ReputationBar score={me?.reputation ?? 0} />
      </View>

      <View style={styles.statsGrid}>
        <StatCard label="Total Earned" value={`$${(me?.total_earned ?? 0).toFixed(2)}`} color="#4CAF50" />
        <StatCard label="Total Spent" value={`$${(me?.total_spent ?? 0).toFixed(2)}`} color="#FF9800" />
        <StatCard label="Tasks Created" value={String(myTasks.length)} />
        <StatCard label="Completed" value={String(completedTasks)} color="#4CAF50" />
        <StatCard label="Open" value={String(openTasks)} color="#2196F3" />
      </View>

      <TouchableOpacity
        style={styles.refreshButton}
        onPress={() => { fetchMe(); fetchMyTasks(); }}
      >
        <Text style={styles.refreshText}>Refresh</Text>
      </TouchableOpacity>

      <View style={styles.infoBox}>
        <Text style={styles.infoText}>
          Your identity is anonymous and device-based. No account needed.
        </Text>
      </View>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: '#000' },
  center: { flex: 1, justifyContent: 'center', alignItems: 'center', backgroundColor: '#000' },
  header: {
    alignItems: 'center',
    padding: 32,
    backgroundColor: '#111',
    borderBottomWidth: 1,
    borderBottomColor: '#222',
  },
  avatar: {
    width: 80,
    height: 80,
    borderRadius: 40,
    backgroundColor: '#333',
    justifyContent: 'center',
    alignItems: 'center',
    marginBottom: 12,
  },
  avatarText: { fontSize: 36, color: '#666' },
  userId: { fontSize: 20, fontWeight: 'bold', color: '#fff', letterSpacing: 2 },
  anonBadge: {
    marginTop: 6,
    fontSize: 12,
    color: '#666',
    backgroundColor: '#222',
    paddingHorizontal: 10,
    paddingVertical: 4,
    borderRadius: 12,
    overflow: 'hidden',
  },
  section: { padding: 16 },
  sectionTitle: { fontSize: 14, color: '#666', marginBottom: 8, textTransform: 'uppercase', letterSpacing: 1 },
  repBarContainer: { flexDirection: 'row', alignItems: 'center', gap: 12 },
  repBarBg: { flex: 1, height: 8, backgroundColor: '#222', borderRadius: 4, overflow: 'hidden' },
  repBarFill: { height: '100%', borderRadius: 4 },
  repScore: { fontSize: 14, fontWeight: '600', width: 60, textAlign: 'right' },
  statsGrid: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    padding: 8,
    gap: 8,
  },
  statCard: {
    flex: 1,
    minWidth: '45%',
    backgroundColor: '#111',
    borderRadius: 12,
    padding: 16,
    margin: 4,
    alignItems: 'center',
    borderWidth: 1,
    borderColor: '#222',
  },
  statValue: { fontSize: 22, fontWeight: 'bold', marginBottom: 4 },
  statLabel: { fontSize: 12, color: '#666', textAlign: 'center' },
  refreshButton: {
    margin: 16,
    padding: 14,
    backgroundColor: '#222',
    borderRadius: 10,
    alignItems: 'center',
  },
  refreshText: { color: '#aaa', fontSize: 14 },
  infoBox: {
    margin: 16,
    padding: 16,
    backgroundColor: '#111',
    borderRadius: 10,
    borderWidth: 1,
    borderColor: '#222',
  },
  infoText: { color: '#555', fontSize: 13, lineHeight: 20, textAlign: 'center' },
});
