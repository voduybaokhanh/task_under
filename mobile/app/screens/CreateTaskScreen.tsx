import React, { useState } from 'react';
import {
  View,
  Text,
  TextInput,
  TouchableOpacity,
  StyleSheet,
  ScrollView,
  Alert,
} from 'react-native';
import { useNavigation } from '@react-navigation/native';
import { useTaskStore } from '../../store/useTaskStore';

function addDays(d: number): string {
  return new Date(Date.now() + d * 86400000).toISOString();
}

function DatePresets({
  label,
  options,
  selected,
  onSelect,
}: {
  label: string;
  options: { label: string; days: number }[];
  selected: number;
  onSelect: (days: number) => void;
}) {
  return (
    <View style={styles.presetContainer}>
      <Text style={styles.presetLabel}>{label}</Text>
      <View style={styles.presetRow}>
        {options.map((o) => (
          <TouchableOpacity
            key={o.days}
            style={[styles.presetChip, selected === o.days && styles.presetChipActive]}
            onPress={() => onSelect(o.days)}
          >
            <Text style={[styles.presetText, selected === o.days && styles.presetTextActive]}>
              {o.label}
            </Text>
          </TouchableOpacity>
        ))}
      </View>
    </View>
  );
}

const CLAIM_OPTIONS = [
  { label: '1 day', days: 1 },
  { label: '3 days', days: 3 },
  { label: '7 days', days: 7 },
  { label: '14 days', days: 14 },
];

const OWNER_OPTIONS = [
  { label: '7 days', days: 7 },
  { label: '14 days', days: 14 },
  { label: '30 days', days: 30 },
  { label: '60 days', days: 60 },
];

export default function CreateTaskScreen() {
  const navigation = useNavigation();
  const { createTask, loading } = useTaskStore();
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [rewardAmount, setRewardAmount] = useState('');
  const [maxClaimants, setMaxClaimants] = useState('1');
  const [claimDays, setClaimDays] = useState(7);
  const [ownerDays, setOwnerDays] = useState(30);

  const handleSubmit = async () => {
    if (!title.trim() || !description.trim() || !rewardAmount) {
      Alert.alert('Missing fields', 'Please fill in title, description, and reward amount.');
      return;
    }
    const reward = parseFloat(rewardAmount);
    if (isNaN(reward) || reward <= 0) {
      Alert.alert('Invalid reward', 'Reward must be a positive number.');
      return;
    }

    try {
      await createTask({
        title: title.trim(),
        description: description.trim(),
        reward_amount: reward,
        max_claimants: parseInt(maxClaimants) || 1,
        claim_deadline: addDays(claimDays),
        owner_deadline: addDays(ownerDays),
      });

      Alert.alert('Task created!', 'Your task is now live.', [
        { text: 'OK', onPress: () => navigation.goBack() },
      ]);
    } catch (error: any) {
      Alert.alert('Error', error.message);
    }
  };

  return (
    <ScrollView style={styles.container} keyboardShouldPersistTaps="handled">
      <View style={styles.form}>
        <Text style={styles.label}>Title *</Text>
        <TextInput
          style={styles.input}
          value={title}
          onChangeText={setTitle}
          placeholder="What needs to be done?"
          placeholderTextColor="#555"
          maxLength={120}
        />

        <Text style={styles.label}>Description *</Text>
        <TextInput
          style={[styles.input, styles.textArea]}
          value={description}
          onChangeText={setDescription}
          placeholder="Describe the task in detail..."
          placeholderTextColor="#555"
          multiline
          numberOfLines={5}
          textAlignVertical="top"
          maxLength={2000}
        />

        <Text style={styles.label}>Reward ($) *</Text>
        <TextInput
          style={styles.input}
          value={rewardAmount}
          onChangeText={setRewardAmount}
          placeholder="10.00"
          placeholderTextColor="#555"
          keyboardType="decimal-pad"
        />

        <Text style={styles.label}>Max Claimants</Text>
        <View style={styles.claimantRow}>
          {['1', '2', '3', '5', '10'].map((n) => (
            <TouchableOpacity
              key={n}
              style={[styles.presetChip, maxClaimants === n && styles.presetChipActive]}
              onPress={() => setMaxClaimants(n)}
            >
              <Text style={[styles.presetText, maxClaimants === n && styles.presetTextActive]}>{n}</Text>
            </TouchableOpacity>
          ))}
        </View>

        <DatePresets
          label="Claim Deadline"
          options={CLAIM_OPTIONS}
          selected={claimDays}
          onSelect={setClaimDays}
        />
        <Text style={styles.deadlineHint}>
          Claimants have until {new Date(Date.now() + claimDays * 86400000).toLocaleDateString()} to claim
        </Text>

        <DatePresets
          label="Completion Deadline"
          options={OWNER_OPTIONS}
          selected={ownerDays}
          onSelect={setOwnerDays}
        />
        <Text style={styles.deadlineHint}>
          Work must be approved by {new Date(Date.now() + ownerDays * 86400000).toLocaleDateString()}
        </Text>

        <TouchableOpacity
          style={[styles.submitButton, loading && styles.submitDisabled]}
          onPress={handleSubmit}
          disabled={loading}
        >
          <Text style={styles.submitText}>{loading ? 'Creating...' : 'Create Task'}</Text>
        </TouchableOpacity>
      </View>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: '#000' },
  form: { padding: 16 },
  label: {
    fontSize: 14,
    fontWeight: '600',
    color: '#aaa',
    marginBottom: 8,
    marginTop: 20,
    textTransform: 'uppercase',
    letterSpacing: 0.5,
  },
  input: {
    backgroundColor: '#111',
    borderWidth: 1,
    borderColor: '#333',
    borderRadius: 10,
    padding: 14,
    color: '#fff',
    fontSize: 15,
  },
  textArea: { height: 120 },
  claimantRow: { flexDirection: 'row', gap: 8, flexWrap: 'wrap' },
  presetContainer: { marginTop: 20 },
  presetLabel: {
    fontSize: 14,
    fontWeight: '600',
    color: '#aaa',
    marginBottom: 10,
    textTransform: 'uppercase',
    letterSpacing: 0.5,
  },
  presetRow: { flexDirection: 'row', gap: 8, flexWrap: 'wrap' },
  presetChip: {
    paddingHorizontal: 14,
    paddingVertical: 8,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: '#333',
    backgroundColor: '#111',
  },
  presetChipActive: { backgroundColor: '#4CAF50', borderColor: '#4CAF50' },
  presetText: { color: '#777', fontSize: 14, fontWeight: '500' },
  presetTextActive: { color: '#fff' },
  deadlineHint: { fontSize: 12, color: '#555', marginTop: 6 },
  submitButton: {
    backgroundColor: '#4CAF50',
    padding: 16,
    borderRadius: 12,
    alignItems: 'center',
    marginTop: 32,
    marginBottom: 32,
  },
  submitDisabled: { opacity: 0.5 },
  submitText: { color: '#fff', fontSize: 16, fontWeight: '700' },
});
