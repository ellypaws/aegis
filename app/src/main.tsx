import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import './index.css'
import App from './App.tsx'
import { SettingsProvider } from './contexts/SettingsContext'
import { ThemeProvider } from './contexts/ThemeContext'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <SettingsProvider>
      <ThemeProvider>
        <BrowserRouter>
          <App />
        </BrowserRouter>
      </ThemeProvider>
    </SettingsProvider>
  </StrictMode>,
)
