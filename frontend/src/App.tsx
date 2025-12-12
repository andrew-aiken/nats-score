import { Outlet } from 'react-router-dom'
import NavBar from './components/NavBar'
import ToastContainer from './components/Toast'
import './App.css'

export default function App() {
  return (
    <div className="app-layout">
      <NavBar />
      <main className="app-content">
        <Outlet />
      </main>
      <ToastContainer />
    </div>
  )
}
