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

export default function CreateTaskScreen() {
  const navigation = useNavigation();
  const { createTask, loading } = useTaskStore();
  const [formData, setFormData] = useState({
    title: '',
    description: '',
    reward_amount: '',
    max_claimants: '',
    claim_deadline: '',
    owner_deadline: '',
  });

  const handleSubmit = async () => {
    if (!formData.title || !formData.description || !formData.reward_amount) {
      Alert.alert('Error', 'Please fill in all required fields');
      return;
    }

    try {
      const claimDeadline = new Date(formData.claim_deadline || Date.now() + 7 * 24 * 60 * 60 * 1000);
      const ownerDeadline = new Date(formData.owner_deadline || Date.now() + 30 * 24 * 60 * 60 * 1000);

      await createTask({
        title: formData.title,
        description: formData.description,
        reward_amount: parseFloat(formData.reward_amount),
        max_claimants: parseInt(formData.max_claimants) || 1,
        claim_deadline: claimDeadline.toISOString(),
        owner_deadline: ownerDeadline.toISOString(),
      });

      Alert.alert('Success', 'Task created successfully', [
        { text: 'OK', onPress: () => navigation.goBack() },
      ]);
    } catch (error: any) {
      Alert.alert('Error', error.message);
    }
  };

  return (
    <ScrollView style={styles.container}>
      <View style={styles.form}>
        <Text style={styles.label}>Title *</Text>
        <TextInput
          style={styles.input}
          value={formData.title}
          onChangeText={(text) => setFormData({ ...formData, title: text })}
          placeholder="Task title"
          placeholderTextColor="#666"
        />

        <Text style={styles.label}>Description *</Text>
        <TextInput
          style={[styles.input, styles.textArea]}
          value={formData.description}
          onChangeText={(text) => setFormData({ ...formData, description: text })}
          placeholder="Task description"
          placeholderTextColor="#666"
          multiline
          numberOfLines={4}
        />

        <Text style={styles.label}>Reward Amount ($) *</Text>
        <TextInput
          style={styles.input}
          value={formData.reward_amount}
          onChangeText={(text) => setFormData({ ...formData, reward_amount: text })}
          placeholder="0.00"
          placeholderTextColor="#666"
          keyboardType="numeric"
        />

        <Text style={styles.label}>Max Claimants</Text>
        <TextInput
          style={styles.input}
          value={formData.max_claimants}
          onChangeText={(text) => setFormData({ ...formData, max_claimants: text })}
          placeholder="1"
          placeholderTextColor="#666"
          keyboardType="numeric"
        />

        <Text style={styles.label}>Claim Deadline (YYYY-MM-DD)</Text>
        <TextInput
          style={styles.input}
          value={formData.claim_deadline}
          onChangeText={(text) => setFormData({ ...formData, claim_deadline: text })}
          placeholder="2024-01-15"
          placeholderTextColor="#666"
        />

        <Text style={styles.label}>Owner Deadline (YYYY-MM-DD)</Text>
        <TextInput
          style={styles.input}
          value={formData.owner_deadline}
          onChangeText={(text) => setFormData({ ...formData, owner_deadline: text })}
          placeholder="2024-01-30"
          placeholderTextColor="#666"
        />

        <TouchableOpacity
          style={[styles.button, loading && styles.buttonDisabled]}
          onPress={handleSubmit}
          disabled={loading}
        >
          <Text style={styles.buttonText}>
            {loading ? 'Creating...' : 'Create Task'}
          </Text>
        </TouchableOpacity>
      </View>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#000',
  },
  form: {
    padding: 16,
  },
  label: {
    fontSize: 16,
    fontWeight: '600',
    color: '#fff',
    marginBottom: 8,
    marginTop: 16,
  },
  input: {
    backgroundColor: '#111',
    borderWidth: 1,
    borderColor: '#333',
    borderRadius: 8,
    padding: 12,
    color: '#fff',
    fontSize: 16,
  },
  textArea: {
    height: 100,
    textAlignVertical: 'top',
  },
  button: {
    backgroundColor: '#333',
    padding: 16,
    borderRadius: 8,
    alignItems: 'center',
    marginTop: 24,
  },
  buttonDisabled: {
    opacity: 0.5,
  },
  buttonText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '600',
  },
});
