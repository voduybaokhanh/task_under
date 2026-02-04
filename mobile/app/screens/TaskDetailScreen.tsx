import React, { useEffect } from 'react';
import {
  View,
  Text,
  ScrollView,
  TouchableOpacity,
  StyleSheet,
  ActivityIndicator,
  Alert,
} from 'react-native';
import { useRoute, useNavigation } from '@react-navigation/native';
import { useTaskStore } from '../../store/useTaskStore';
import AsyncStorage from '@react-native-async-storage/async-storage';

export default function TaskDetailScreen() {
  const route = useRoute();
  const navigation = useNavigation();
  const { taskId } = route.params as { taskId: string };
  const { selectedTask, claims, loading, fetchTask, fetchClaims, claimTask, submitCompletion } =
    useTaskStore();
  const [userId, setUserId] = React.useState<string | null>(null);

  useEffect(() => {
    AsyncStorage.getItem('device_id').then(setUserId);
    fetchTask(taskId);
    fetchClaims(taskId);
  }, [taskId]);

  const handleClaim = async () => {
    try {
      await claimTask(taskId);
      Alert.alert('Success', 'Task claimed successfully');
    } catch (error: any) {
      Alert.alert('Error', error.message);
    }
  };

  const handleSubmitCompletion = async (claimId: string) => {
    Alert.prompt(
      'Submit Completion',
      'Enter completion text:',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Submit',
          onPress: async (text) => {
            if (text) {
              try {
                await submitCompletion(claimId, text);
                Alert.alert('Success', 'Completion submitted');
              } catch (error: any) {
                Alert.alert('Error', error.message);
              }
            }
          },
        },
      ]
    );
  };

  if (loading && !selectedTask) {
    return (
      <View style={styles.container}>
        <ActivityIndicator size="large" />
      </View>
    );
  }

  if (!selectedTask) {
    return (
      <View style={styles.container}>
        <Text style={styles.error}>Task not found</Text>
      </View>
    );
  }

  const isOwner = selectedTask.owner_id === userId;
  const userClaim = claims.find((c) => c.claimer_id === userId);
  const canClaim = !isOwner && !userClaim && selectedTask.status === 'open';

  return (
    <ScrollView style={styles.container}>
      <View style={styles.content}>
        <Text style={styles.title}>{selectedTask.title}</Text>
        <Text style={styles.reward}>${selectedTask.reward_amount.toFixed(2)}</Text>
        <Text style={styles.description}>{selectedTask.description}</Text>

        <View style={styles.meta}>
          <Text style={styles.metaText}>
            Max Claimants: {selectedTask.max_claimants}
          </Text>
          <Text style={styles.metaText}>
            Claim Deadline: {new Date(selectedTask.claim_deadline).toLocaleString()}
          </Text>
          <Text style={styles.metaText}>
            Owner Deadline: {new Date(selectedTask.owner_deadline).toLocaleString()}
          </Text>
          <Text style={styles.metaText}>Status: {selectedTask.status}</Text>
          <Text style={styles.metaText}>
            Current Claims: {claims.length}/{selectedTask.max_claimants}
          </Text>
        </View>

        {canClaim && (
          <TouchableOpacity style={styles.button} onPress={handleClaim}>
            <Text style={styles.buttonText}>Claim Task</Text>
          </TouchableOpacity>
        )}

        {userClaim && (
          <View style={styles.claimSection}>
            <Text style={styles.sectionTitle}>Your Claim</Text>
            <Text style={styles.claimStatus}>Status: {userClaim.status}</Text>
            {userClaim.status === 'pending' && !userClaim.submitted_at && (
              <TouchableOpacity
                style={styles.button}
                onPress={() => handleSubmitCompletion(userClaim.id)}
              >
                <Text style={styles.buttonText}>Submit Completion</Text>
              </TouchableOpacity>
            )}
            {userClaim.submitted_at && (
              <TouchableOpacity
                style={styles.button}
                onPress={() =>
                  navigation.navigate('Chat' as never, { taskId, claimerId: userClaim.claimer_id } as never)
                }
              >
                <Text style={styles.buttonText}>Open Chat</Text>
              </TouchableOpacity>
            )}
          </View>
        )}

        {isOwner && claims.length > 0 && (
          <View style={styles.claimsSection}>
            <Text style={styles.sectionTitle}>Claims ({claims.length})</Text>
            {claims.map((claim) => (
              <View key={claim.id} style={styles.claimCard}>
                <Text style={styles.claimText}>Status: {claim.status}</Text>
                {claim.completion_text && (
                  <Text style={styles.completionText}>{claim.completion_text}</Text>
                )}
                <View style={styles.claimActions}>
                  {claim.status === 'pending' && claim.submitted_at && (
                    <>
                      <TouchableOpacity
                        style={[styles.button, styles.approveButton]}
                        onPress={() => {
                          useTaskStore.getState().approveClaim(claim.id);
                        }}
                      >
                        <Text style={styles.buttonText}>Approve</Text>
                      </TouchableOpacity>
                      <TouchableOpacity
                        style={[styles.button, styles.rejectButton]}
                        onPress={() => {
                          useTaskStore.getState().rejectClaim(claim.id);
                        }}
                      >
                        <Text style={styles.buttonText}>Reject</Text>
                      </TouchableOpacity>
                      <TouchableOpacity
                        style={styles.button}
                        onPress={() =>
                          navigation.navigate('Chat' as never, { taskId, claimerId: claim.claimer_id } as never)
                        }
                      >
                        <Text style={styles.buttonText}>Chat</Text>
                      </TouchableOpacity>
                    </>
                  )}
                </View>
              </View>
            ))}
          </View>
        )}
      </View>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#000',
  },
  content: {
    padding: 16,
  },
  title: {
    fontSize: 24,
    fontWeight: 'bold',
    color: '#fff',
    marginBottom: 8,
  },
  reward: {
    fontSize: 28,
    fontWeight: 'bold',
    color: '#4CAF50',
    marginBottom: 16,
  },
  description: {
    fontSize: 16,
    color: '#aaa',
    marginBottom: 16,
    lineHeight: 24,
  },
  meta: {
    backgroundColor: '#111',
    padding: 16,
    borderRadius: 8,
    marginBottom: 16,
  },
  metaText: {
    fontSize: 14,
    color: '#aaa',
    marginBottom: 4,
  },
  button: {
    backgroundColor: '#333',
    padding: 16,
    borderRadius: 8,
    alignItems: 'center',
    marginBottom: 16,
  },
  buttonText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '600',
  },
  approveButton: {
    backgroundColor: '#4CAF50',
  },
  rejectButton: {
    backgroundColor: '#f44336',
  },
  claimSection: {
    backgroundColor: '#111',
    padding: 16,
    borderRadius: 8,
    marginBottom: 16,
  },
  sectionTitle: {
    fontSize: 18,
    fontWeight: 'bold',
    color: '#fff',
    marginBottom: 8,
  },
  claimStatus: {
    fontSize: 14,
    color: '#aaa',
    marginBottom: 8,
  },
  claimsSection: {
    marginTop: 16,
  },
  claimCard: {
    backgroundColor: '#111',
    padding: 16,
    borderRadius: 8,
    marginBottom: 8,
  },
  claimText: {
    fontSize: 14,
    color: '#aaa',
    marginBottom: 4,
  },
  completionText: {
    fontSize: 14,
    color: '#fff',
    marginBottom: 8,
  },
  claimActions: {
    flexDirection: 'row',
    gap: 8,
  },
  error: {
    color: '#ff4444',
    textAlign: 'center',
    marginTop: 50,
  },
});
