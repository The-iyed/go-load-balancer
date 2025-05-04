import { FC, useEffect, useState } from 'react';
import { Box, Paper, Typography, CircularProgress, Chip } from '@mui/material';
import { 
  PieChart, Pie, Cell, XAxis, YAxis, 
  CartesianGrid, Tooltip, ResponsiveContainer, LineChart, Line 
} from 'recharts';
import { fetchStats, fetchTimeSeries } from '../api';
import { Stats, ChartDataPoint, BackendTimeSeriesData } from '../types';

// Dashboard status card
interface StatusCardProps {
  title: string;
  value: string | number;
  subtitle?: string;
  color?: string;
}

const StatusCard: FC<StatusCardProps> = ({ title, value, subtitle, color = '#1565C0' }) => (
  <Paper
    elevation={2}
    sx={{
      p: 3,
      height: '100%',
      display: 'flex',
      flexDirection: 'column',
      borderTop: `4px solid ${color}`,
    }}
  >
    <Typography variant="h6" component="div" color="text.secondary" gutterBottom>
      {title}
    </Typography>
    <Typography variant="h4" component="div" sx={{ my: 1, fontWeight: 500 }}>
      {value}
    </Typography>
    {subtitle && (
      <Typography variant="body2" color="text.secondary">
        {subtitle}
      </Typography>
    )}
  </Paper>
);

// Colors for charts
const CHART_COLORS = ['#0088FE', '#00C49F', '#FFBB28', '#FF8042', '#8884D8', '#82CA9D'];

const Dashboard: FC = () => {
  const [stats, setStats] = useState<Stats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [timeSeriesData, setTimeSeriesData] = useState<BackendTimeSeriesData[]>([]);

  useEffect(() => {
    // Fetch initial stats
    const fetchInitialData = async () => {
      try {
        const data = await fetchStats();
        setStats(data);
        setLoading(false);
        
        // Fetch time series data for each backend
        if (data.backends.length > 0) {
          const promises = data.backends.map(backend => 
            fetchTimeSeries(backend.url, '1h')
              .then(response => ({
                name: getBackendName(backend.url),
                data: response.data
              }))
          );
          
          const results = await Promise.all(promises);
          setTimeSeriesData(results);
        }
      } catch (err) {
        setError('Failed to fetch load balancer statistics');
        setLoading(false);
        console.error(err);
      }
    };

    fetchInitialData();

    // Set up polling every 5 seconds
    const interval = setInterval(async () => {
      try {
        const data = await fetchStats();
        setStats(data);
      } catch (err) {
        console.error('Error refreshing stats:', err);
      }
    }, 5000);

    return () => clearInterval(interval);
  }, []);

  // Helper to extract backend name from URL
  const getBackendName = (url: string): string => {
    try {
      const urlObj = new URL(url);
      return `${urlObj.hostname}:${urlObj.port}`;
    } catch (e) {
      return url;
    }
  };

  // Prepare data for pie chart
  const preparePieData = (): ChartDataPoint[] => {
    if (!stats) return [];
    
    return stats.backends.map(backend => ({
      name: getBackendName(backend.url),
      value: backend.requestCount,
    }));
  };

  if (loading) {
    return (
      <Box 
        sx={{ 
          display: 'flex', 
          justifyContent: 'center', 
          alignItems: 'center',
          height: '100%' 
        }}
      >
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Box sx={{ p: 3 }}>
        <Typography color="error">{error}</Typography>
      </Box>
    );
  }

  if (!stats) {
    return (
      <Box sx={{ p: 3 }}>
        <Typography>No data available</Typography>
      </Box>
    );
  }

  return (
    <Box sx={{ p: 3 }}>
      <Box sx={{ mb: 4 }}>
        <Typography variant="h4" component="h1" gutterBottom>
          Load Balancer Dashboard
        </Typography>
        <Typography variant="subtitle1" color="text.secondary" gutterBottom>
          Monitor your load balancer performance in real-time
        </Typography>
      </Box>

      {/* Status Cards */}
      <Box sx={{ 
        display: 'flex', 
        flexWrap: 'wrap', 
        gap: 3, 
        mb: 3 
      }}>
        <Box sx={{ flex: '1 1 220px' }}>
          <StatusCard 
            title="Total Requests" 
            value={stats.totalRequests} 
            subtitle="Since startup" 
            color="#1565C0"
          />
        </Box>
        <Box sx={{ flex: '1 1 220px' }}>
          <StatusCard 
            title="Algorithm" 
            value={stats.method} 
            subtitle="Load balancing method" 
            color="#00695C"
          />
        </Box>
        <Box sx={{ flex: '1 1 220px' }}>
          <StatusCard 
            title="Persistence" 
            value={stats.persistenceType} 
            subtitle="Session handling" 
            color="#EF6C00"
          />
        </Box>
        <Box sx={{ flex: '1 1 220px' }}>
          <StatusCard 
            title="Uptime" 
            value={stats.uptime.split('.')[0]} // Format to remove milliseconds
            subtitle="Server running time" 
            color="#5E35B1"
          />
        </Box>
      </Box>

      {/* Charts */}
      <Box sx={{ 
        display: 'flex', 
        flexWrap: 'wrap', 
        gap: 3 
      }}>
        {/* Backends Status */}
        <Box sx={{ flex: '1 1 450px' }}>
          <Paper elevation={2} sx={{ p: 2, height: '100%' }}>
            <Typography variant="h6" gutterBottom>Backend Status</Typography>
            <Box sx={{ overflowX: 'auto' }}>
              <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                <thead>
                  <tr style={{ borderBottom: '1px solid #e0e0e0' }}>
                    <th style={{ textAlign: 'left', padding: '8px' }}>Server</th>
                    <th style={{ textAlign: 'left', padding: '8px' }}>Status</th>
                    <th style={{ textAlign: 'right', padding: '8px' }}>Requests</th>
                    <th style={{ textAlign: 'right', padding: '8px' }}>Errors</th>
                    <th style={{ textAlign: 'right', padding: '8px' }}>Load %</th>
                  </tr>
                </thead>
                <tbody>
                  {stats.backends.map((backend, idx) => (
                    <tr key={idx} style={{ borderBottom: '1px solid #f5f5f5' }}>
                      <td style={{ padding: '8px' }}>{getBackendName(backend.url)}</td>
                      <td style={{ padding: '8px' }}>
                        <Chip 
                          label={backend.alive ? "Healthy" : "Down"} 
                          size="small"
                          color={backend.alive ? "success" : "error"} 
                        />
                      </td>
                      <td style={{ textAlign: 'right', padding: '8px' }}>{backend.requestCount}</td>
                      <td style={{ textAlign: 'right', padding: '8px' }}>{backend.errorCount}</td>
                      <td style={{ textAlign: 'right', padding: '8px' }}>
                        {backend.loadPercentage.toFixed(1)}%
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </Box>
          </Paper>
        </Box>

        {/* Request Distribution Pie Chart */}
        <Box sx={{ flex: '1 1 450px' }}>
          <Paper elevation={2} sx={{ p: 2, height: '100%' }}>
            <Typography variant="h6" gutterBottom>Request Distribution</Typography>
            <ResponsiveContainer width="100%" height={300}>
              <PieChart>
                <Pie
                  data={preparePieData()}
                  cx="50%"
                  cy="50%"
                  labelLine={false}
                  outerRadius={100}
                  fill="#8884d8"
                  dataKey="value"
                  label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`}
                >
                  {preparePieData().map((_, index) => (
                    <Cell key={`cell-${index}`} fill={CHART_COLORS[index % CHART_COLORS.length]} />
                  ))}
                </Pie>
                <Tooltip formatter={(value) => [`${value} requests`, 'Count']} />
              </PieChart>
            </ResponsiveContainer>
          </Paper>
        </Box>

        {/* Traffic Over Time Chart */}
        <Box sx={{ flex: '1 1 100%' }}>
          <Paper elevation={2} sx={{ p: 2 }}>
            <Typography variant="h6" gutterBottom>Traffic Over Time (Last Hour)</Typography>
            <ResponsiveContainer width="100%" height={300}>
              <LineChart
                margin={{ top: 5, right: 30, left: 20, bottom: 5 }}
              >
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis 
                  dataKey="time" 
                  type="number"
                  scale="time"
                  domain={['auto', 'auto']}
                  tickFormatter={(unixTime) => new Date(unixTime).toLocaleTimeString()}
                />
                <YAxis />
                <Tooltip 
                  labelFormatter={(value) => new Date(value).toLocaleString()}
                  formatter={(value) => [`${value} requests`, 'Count']} 
                />
                {timeSeriesData.map((s, index) => (
                  <Line 
                    key={s.name}
                    type="monotone" 
                    data={s.data}
                    dataKey="value" 
                    name={s.name}
                    stroke={CHART_COLORS[index % CHART_COLORS.length]} 
                    activeDot={{ r: 8 }} 
                  />
                ))}
              </LineChart>
            </ResponsiveContainer>
          </Paper>
        </Box>
      </Box>
    </Box>
  );
};

export default Dashboard; 