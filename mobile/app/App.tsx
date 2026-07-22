import React, { useEffect } from 'react';
import { NavigationContainer } from '@react-navigation/native';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import { createBottomTabNavigator } from '@react-navigation/bottom-tabs';
import { StatusBar } from 'expo-status-bar';
import { Ionicons } from '@expo/vector-icons';

import TaskListScreen from './screens/TaskListScreen';
import TaskDetailScreen from './screens/TaskDetailScreen';
import CreateTaskScreen from './screens/CreateTaskScreen';
import MyTasksScreen from './screens/MyTasksScreen';
import ChatScreen from './screens/ChatScreen';
import ProfileScreen from './screens/ProfileScreen';
import { registerForPushNotifications } from '../services/notifications';
import { publishPublicKey } from '../services/keys';
import { connectRealtime, disconnectRealtime } from '../services/realtime';
import { cardPaymentsEnabled, stripePublishableKey } from '../services/payment';
import { StripeProvider } from '@stripe/stripe-react-native';

const Stack = createNativeStackNavigator();
const Tab = createBottomTabNavigator();

const screenOptions = {
  headerStyle: { backgroundColor: '#111' },
  headerTintColor: '#fff',
  headerTitleStyle: { fontWeight: 'bold' as const },
};

function TasksTab() {
  return (
    <Stack.Navigator screenOptions={screenOptions}>
      <Stack.Screen name="TaskList" component={TaskListScreen} options={{ title: 'Tasks' }} />
      <Stack.Screen name="TaskDetail" component={TaskDetailScreen} options={{ title: 'Task Details' }} />
      <Stack.Screen name="CreateTask" component={CreateTaskScreen} options={{ title: 'Create Task' }} />
      <Stack.Screen name="Chat" component={ChatScreen} options={{ title: 'Chat' }} />
    </Stack.Navigator>
  );
}

function MyTasksTab() {
  return (
    <Stack.Navigator screenOptions={screenOptions}>
      <Stack.Screen name="MyTasks" component={MyTasksScreen} options={{ title: 'My Tasks' }} />
      <Stack.Screen name="TaskDetail" component={TaskDetailScreen} options={{ title: 'Task Details' }} />
      <Stack.Screen name="Chat" component={ChatScreen} options={{ title: 'Chat' }} />
    </Stack.Navigator>
  );
}

function ProfileTab() {
  return (
    <Stack.Navigator screenOptions={screenOptions}>
      <Stack.Screen name="Profile" component={ProfileScreen} options={{ title: 'Profile' }} />
    </Stack.Navigator>
  );
}

export default function App() {
  useEffect(() => {
    registerForPushNotifications();
    // Publish our E2EE public key so others can start an encrypted chat.
    publishPublicKey().catch((error) => console.warn('Publishing public key failed', error));
    connectRealtime().catch((error) => console.warn('Realtime connection failed', error));

    return disconnectRealtime;
  }, []);

  const app = (
    <NavigationContainer>
      <StatusBar style="light" />
      <Tab.Navigator
        screenOptions={({ route }) => ({
          headerShown: false,
          tabBarStyle: {
            backgroundColor: '#111',
            borderTopColor: '#222',
          },
          tabBarActiveTintColor: '#4CAF50',
          tabBarInactiveTintColor: '#555',
          tabBarIcon: ({ focused, color, size }) => {
            let iconName: keyof typeof Ionicons.glyphMap;
            if (route.name === 'Tasks') {
              iconName = focused ? 'list' : 'list-outline';
            } else if (route.name === 'MyTasks') {
              iconName = focused ? 'briefcase' : 'briefcase-outline';
            } else {
              iconName = focused ? 'person' : 'person-outline';
            }
            return <Ionicons name={iconName} size={size} color={color} />;
          },
        })}
      >
        <Tab.Screen name="Tasks" component={TasksTab} options={{ title: 'Explore' }} />
        <Tab.Screen name="MyTasks" component={MyTasksTab} options={{ title: 'My Tasks' }} />
        <Tab.Screen name="ProfileTab" component={ProfileTab} options={{ title: 'Profile' }} />
      </Tab.Navigator>
    </NavigationContainer>
  );

  // Without a publishable key the Stripe provider has nothing to do, and the
  // app runs on simulated escrow.
  return cardPaymentsEnabled ? (
    <StripeProvider publishableKey={stripePublishableKey}>{app}</StripeProvider>
  ) : (
    app
  );
}
