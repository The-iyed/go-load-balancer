import { FC } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { 
  Box, 
  Drawer, 
  List, 
  ListItem, 
  ListItemButton, 
  ListItemIcon, 
  ListItemText,
  Divider,
  Typography
} from '@mui/material';
import { 
  Dashboard as DashboardIcon, 
  Storage as StorageIcon, 
  Route as RouteIcon,
  Settings as SettingsIcon,
  Equalizer as StatsIcon
} from '@mui/icons-material';

// Sidebar width
const drawerWidth = 240;

interface SidebarProps {
  open: boolean;
  onClose?: () => void;
  variant?: "permanent" | "persistent" | "temporary";
}

const Sidebar: FC<SidebarProps> = ({ open, onClose, variant = "permanent" }) => {
  const location = useLocation();
  const isActive = (path: string) => location.pathname === path;

  const menuItems = [
    { text: 'Dashboard', icon: <DashboardIcon />, path: '/' },
    { text: 'Backends', icon: <StorageIcon />, path: '/backends' },
    { text: 'Routes', icon: <RouteIcon />, path: '/routes' },
    { text: 'Statistics', icon: <StatsIcon />, path: '/statistics' },
    { text: 'Settings', icon: <SettingsIcon />, path: '/settings' },
  ];

  return (
    <Drawer
      variant={variant}
      open={open}
      onClose={onClose}
      sx={{
        width: drawerWidth,
        flexShrink: 0,
        '& .MuiDrawer-paper': {
          width: drawerWidth,
          boxSizing: 'border-box',
          background: '#0B2447',
          color: 'white',
        },
      }}
    >
      <Box sx={{ p: 2, textAlign: 'center' }}>
        <Typography variant="h6" component="div" sx={{ fontWeight: 'bold' }}>
          Go Load Balancer
        </Typography>
        <Typography variant="subtitle2" component="div" sx={{ opacity: 0.7 }}>
          Admin Dashboard
        </Typography>
      </Box>
      <Divider sx={{ backgroundColor: 'rgba(255,255,255,0.1)' }} />
      <List>
        {menuItems.map((item) => (
          <ListItem key={item.text} disablePadding>
            <ListItemButton 
              component={Link} 
              to={item.path}
              selected={isActive(item.path)}
              sx={{
                '&.Mui-selected': {
                  backgroundColor: 'rgba(255,255,255,0.1)',
                  '&:hover': {
                    backgroundColor: 'rgba(255,255,255,0.15)',
                  },
                },
                '&:hover': {
                  backgroundColor: 'rgba(255,255,255,0.05)',
                },
              }}
            >
              <ListItemIcon sx={{ color: 'white', minWidth: '40px' }}>
                {item.icon}
              </ListItemIcon>
              <ListItemText primary={item.text} />
            </ListItemButton>
          </ListItem>
        ))}
      </List>
    </Drawer>
  );
};

export default Sidebar; 