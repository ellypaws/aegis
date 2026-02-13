
function App() {
  return (
    <div className="min-h-screen bg-[#0a0a0a] flex items-center justify-center p-4">
      <div className="relative group">
        {/* Glow effect */}
        <div className="absolute -inset-1 bg-gradient-to-r from-blue-600 to-violet-600 rounded-lg blur opacity-25 group-hover:opacity-50 transition duration-1000 group-hover:duration-200"></div>

        <div className="relative px-8 py-6 bg-black rounded-lg leading-none flex items-center divide-x divide-gray-600">
          <span className="flex items-center space-x-5">
            <svg xmlns="http://www.w3.org/2000/svg" className="h-6 w-6 text-blue-500 -rotate-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M7 20l4-16m2 16l4-16" />
            </svg>
            <span className="pr-6 text-gray-100 font-medium tracking-tight">Initial commit</span>
          </span>
          <span className="pl-6 text-blue-400 group-hover:text-blue-300 transition duration-200 uppercase text-xs tracking-[0.2em] font-bold">
            v0.0.1
          </span>
        </div>
      </div>

      {/* Subtle background flair */}
      <div className="fixed top-0 left-0 w-full h-full pointer-events-none -z-10 overflow-hidden">
        <div className="absolute top-[-10%] left-[-10%] w-[40%] h-[40%] bg-blue-500/10 rounded-full blur-[120px]"></div>
        <div className="absolute bottom-[-10%] right-[-10%] w-[40%] h-[40%] bg-violet-500/10 rounded-full blur-[120px]"></div>
      </div>
    </div>
  )
}

export default App
