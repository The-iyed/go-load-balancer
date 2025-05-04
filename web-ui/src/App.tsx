import { useState } from 'react'
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom'
import { 
  CssBaseline, 
  Box, 
  ThemeProvider, 
  createTheme, 
  useMediaQuery, 
  IconButton,
  AppBar,
  Toolbar,
  Typography,
} from '@mui/material'
import MenuIcon from '@mui/icons-material/Menu'

// Import components
import Sidebar from './components/Sidebar'
import Dashboard from './components/Dashboard'

// Simple placeholder for routes that aren't implemented yet
const PlaceholderPage = ({ title }: { title: string }) => (
  <Box sx={{ p: 3 }}>
    <Typography variant="h4" component="h1" gutterBottom>
      {title}
    </Typography>
    <Typography variant="body1">
      This page is under construction. Check back soon!
    </Typography>
  </Box>
)

function App() {
  const prefersDarkMode = useMediaQuery('(prefers-color-scheme: dark)')
  const [mobileOpen, setMobileOpen] = useState(false)

  // Create theme based on user preference
  const theme = createTheme({
    palette: {
      mode: prefersDarkMode ? 'dark' : 'light',
      primary: {
        main: '#1976d2',
      },
      secondary: {
        main: '#f50057',
      },
    },
  })

  const handleDrawerToggle = () => {
    setMobileOpen(!mobileOpen)
  }

  return (
    <ThemeProvider theme={theme}>
      <Router>
        <Box sx={{ display: 'flex' }}>
          <CssBaseline />
          
          {/* AppBar */}
          <AppBar
            position="fixed"
            sx={{
              zIndex: (theme) => theme.zIndex.drawer + 1,
              ml: { sm: '240px' },
              width: { sm: `calc(100% - 240px)` },
            }}
          >
            <Toolbar>
              <IconButton
                color="inherit"
                aria-label="open drawer"
                edge="start"
                onClick={handleDrawerToggle}
                sx={{ mr: 2, display: { sm: 'none' } }}
              >
                <MenuIcon />
              </IconButton>
              <Typography variant="h6" noWrap component="div">
                Load Balancer Dashboard
              </Typography>
            </Toolbar>
          </AppBar>
          
          {/* Sidebar */}
          <Box
            component="nav"
            sx={{ width: { sm: 240 }, flexShrink: { sm: 0 } }}
          >
            {/* Mobile drawer */}
            <Sidebar
              variant="temporary"
              open={mobileOpen}
              onClose={handleDrawerToggle}
            />
            
            {/* Desktop drawer */}
            <Sidebar
              variant="permanent"
              open={true}
            />
          </Box>
          
          {/* Main content */}
          <Box 
            component="main" 
            sx={{ 
              flexGrow: 1, 
              p: 3, 
              width: { sm: `calc(100% - 240px)` },
              marginTop: '64px', // AppBar height
              minHeight: 'calc(100vh - 64px)', // Full height minus AppBar
            }}
          >
            <Routes>
              <Route path="/" element={<Dashboard />} />
              <Route path="/backends" element={<PlaceholderPage title="Backends Management" />} />
              <Route path="/routes" element={<PlaceholderPage title="Path Router Configuration" />} />
              <Route path="/statistics" element={<PlaceholderPage title="Statistics" />} />
              <Route path="/settings" element={<PlaceholderPage title="Settings" />} />
            </Routes>
          </Box>
        </Box>
      </Router>
    </ThemeProvider>
  )
}

export default App
