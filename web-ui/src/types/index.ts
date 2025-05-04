// Types for the API responses
export interface Stats {
  backends: BackendStats[];
  method: string;
  totalRequests: number;
  persistenceType: string;
  routeStats?: Record<string, string>;
  startTime: string;
  uptime: string;
}

export interface BackendStats {
  url: string;
  alive: boolean;
  weight: number;
  requestCount: number;
  errorCount: number;
  loadPercentage: number;
  responseTimeAvg: number;
}

// Chart data types
export interface ChartDataPoint {
  name: string;
  value: number;
}

export interface TimeSeriesPoint {
  time: number;
  value: number;
}

export interface BackendTimeSeriesData {
  name: string;
  data: TimeSeriesPoint[];
} 