import axios from 'axios';
import { Stats } from '../types';

// Set base URL depending on environment
const apiBaseUrl = import.meta.env.DEV 
  ? 'http://localhost:8081/api'  // Development
  : '/api';                      // Production (relative path)

const api = axios.create({
  baseURL: apiBaseUrl,
  timeout: 5000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// API functions
export const fetchStats = async (): Promise<Stats> => {
  const response = await api.get<Stats>('/stats');
  return response.data;
};

// Time-series data for dashboard
export const fetchTimeSeries = async (_unused: string, _period: string): Promise<any> => {
  // This is a placeholder. In a real implementation, we would fetch time-series data
  // from the server. For now, we'll return mock data from the client.
  return Promise.resolve({
    data: generateMockTimeSeriesData(),
  });
};

// Helper function to generate mock time-series data
const generateMockTimeSeriesData = () => {
  const now = Date.now();
  const data = [];
  
  // Generate data points for the last hour
  for (let i = 0; i < 60; i++) {
    data.push({
      time: now - (60 - i) * 60000, // 60 minutes ago to now
      value: Math.floor(Math.random() * 100), // Random value between 0-100
    });
  }
  
  return data;
}; 