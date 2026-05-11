package handlers

const loginPage = `<!DOCTYPE html>
<html lang="en" class="dark">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Login - 4rch3.io</title>
    <script src="https://unpkg.com/htmx.org@2.0.4"></script>
    <script src="https://cdn.tailwindcss.com"></script>
    <script>
        tailwind.config = {
            darkMode: 'class',
            theme: {
                extend: {
                    colors: {
                        neon: {
                            cyan: '#00f0ff',
                            purple: '#a855f7',
                            pink: '#ec4899',
                        }
                    }
                }
            }
        }
    </script>
    <style>
        @keyframes glow {
            0%, 100% { box-shadow: 0 0 20px rgba(0, 240, 255, 0.3); }
            50% { box-shadow: 0 0 40px rgba(0, 240, 255, 0.6); }
        }
        .glow-border { animation: glow 3s ease-in-out infinite; }
        body { background: #0a0a0f; }
        .grid-bg {
            background-image:
                linear-gradient(rgba(0, 240, 255, 0.03) 1px, transparent 1px),
                linear-gradient(90deg, rgba(0, 240, 255, 0.03) 1px, transparent 1px);
            background-size: 40px 40px;
        }
    </style>
</head>
<body class="grid-bg min-h-screen flex items-center justify-center p-4">
    <div class="w-full max-w-md">
        <div class="text-center mb-8">
            <h1 class="text-4xl font-bold bg-gradient-to-r from-neon-cyan via-neon-purple to-neon-pink bg-clip-text text-transparent">4rch3.io</h1>
            <p class="text-gray-500 mt-2 text-sm">homelab operations portal</p>
        </div>

        <div class="bg-gray-900/80 backdrop-blur-xl rounded-2xl p-8 border border-gray-800 glow-border">
            <form hx-post="/login" hx-target="this" hx-swap="outerHTML" class="space-y-6">
                <div id="login-error" class="text-red-400 text-sm text-center"></div>

                <div>
                    <label class="block text-sm font-medium text-gray-400 mb-2">Username</label>
                    <input type="text" name="username" required
                           class="w-full px-4 py-3 bg-gray-800/50 border border-gray-700 rounded-xl text-gray-100 placeholder-gray-600 focus:border-neon-cyan focus:ring-1 focus:ring-neon-cyan outline-none transition-all"
                           placeholder="enter username">
                </div>

                <div>
                    <label class="block text-sm font-medium text-gray-400 mb-2">Password</label>
                    <input type="password" name="password" required
                           class="w-full px-4 py-3 bg-gray-800/50 border border-gray-700 rounded-xl text-gray-100 placeholder-gray-600 focus:border-neon-cyan focus:ring-1 focus:ring-neon-cyan outline-none transition-all"
                           placeholder="enter password">
                </div>

                <button type="submit"
                        class="w-full py-3 px-4 bg-gradient-to-r from-neon-cyan to-neon-purple text-black font-semibold rounded-xl hover:opacity-90 transition-all duration-200">
                    Sign In
                </button>
            </form>
        </div>

        <p class="text-center text-gray-600 text-xs mt-6">
            authorized personnel only &middot; all activity is logged
        </p>
    </div>
</body>
</html>`
