/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        // Neutral ramp for the authenticated product theme. Semantic
        // workspace roles remain the source of truth for shell surfaces;
        // this ramp keeps legacy utility-based screens visually consistent.
        gray: {
          50: '#f7f7f8',
          100: '#ececec',
          200: '#e5e5e5',
          300: '#d4d4d4',
          400: '#8e8e8e',
          500: '#737373',
          600: '#5d5d5d',
          700: '#424242',
          800: '#212121',
          900: '#0d0d0d',
          950: '#050505'
        },
        // Snow Clay primary interaction — violet. Ice blue remains a scoped
        // brand/information role in luoxue-clay-tokens.css.
        primary: {
          50: '#f5f3ff',
          100: '#ede9fe',
          200: '#ddd6fe',
          300: '#c4b5fd',
          400: '#a78bfa',
          500: '#8b5cf6',
          600: '#7c3aed',
          700: '#6d28d9',
          800: '#5b21b6',
          900: '#4c1d95',
          950: '#2e1065'
        },
        // 辅助色 - 深蓝灰
        accent: {
          50: '#f8fafc',
          100: '#f1f5f9',
          200: '#e2e8f0',
          300: '#cbd5e1',
          400: '#94a3b8',
          500: '#64748b',
          600: '#475569',
          700: '#334155',
          800: '#1e293b',
          900: '#0f172a',
          950: '#020617'
        },
        // ChatGPT-like dark neutral ramp for legacy dark-* utilities.
        dark: {
          50: '#f7f7f8',
          100: '#ececec',
          200: '#d4d4d4',
          300: '#b4b4b4',
          400: '#8e8e8e',
          500: '#737373',
          600: '#565656',
          700: '#424242',
          800: '#2f2f2f',
          900: '#212121',
          950: '#171717'
        },
        workspace: {
          canvas: 'var(--workspace-canvas)',
          sidebar: 'var(--workspace-sidebar-surface)',
          surface: 'var(--workspace-surface)',
          card: 'var(--workspace-card-surface)',
          popup: 'var(--workspace-popup-surface)',
          subtle: 'var(--workspace-surface-subtle)',
          hover: 'var(--workspace-hover)',
          selected: 'var(--workspace-selected)',
          divider: 'var(--workspace-divider)',
          border: 'var(--workspace-border)',
          'border-strong': 'var(--workspace-border-strong)',
          text: 'var(--workspace-text)',
          'text-secondary': 'var(--workspace-text-secondary)',
          muted: 'var(--workspace-text-muted)',
          action: 'var(--workspace-action)',
          'action-hover': 'var(--workspace-action-hover)',
          'action-soft': 'var(--workspace-action-soft)'
        }
      },
      fontFamily: {
        workspace: ['var(--workspace-font-ui)'],
        sans: [
          'system-ui',
          '-apple-system',
          'BlinkMacSystemFont',
          'Segoe UI',
          'Roboto',
          'Helvetica Neue',
          'Arial',
          'PingFang SC',
          'Hiragino Sans GB',
          'Microsoft YaHei',
          'sans-serif'
        ],
        mono: ['ui-monospace', 'SFMono-Regular', 'Menlo', 'Monaco', 'Consolas', 'monospace']
      },
      boxShadow: {
        glass: '0 16px 48px rgba(15, 23, 42, 0.08)',
        'glass-sm': '0 8px 24px rgba(15, 23, 42, 0.06)',
        glow: '0 8px 24px rgba(124, 58, 237, 0.16)',
        'glow-lg': '0 16px 40px rgba(124, 58, 237, 0.2)',
        card: '0 1px 2px rgba(15, 23, 42, 0.04)',
        'card-hover': '0 8px 24px rgba(15, 23, 42, 0.06)',
        'inner-glow': 'inset 0 1px 0 rgba(255, 255, 255, 0.1)'
      },
      backgroundImage: {
        'gradient-radial': 'radial-gradient(var(--tw-gradient-stops))',
        'gradient-primary': 'linear-gradient(135deg, #8b5cf6 0%, #5b21b6 100%)',
        'gradient-dark': 'linear-gradient(135deg, #1e293b 0%, #0f172a 100%)',
        'gradient-glass':
          'linear-gradient(135deg, rgba(255,255,255,0.1) 0%, rgba(255,255,255,0.05) 100%)',
        'mesh-gradient': 'none'
      },
      animation: {
        'fade-in': 'fadeIn 0.3s ease-out',
        'slide-up': 'slideUp 0.3s ease-out',
        'slide-down': 'slideDown 0.3s ease-out',
        'slide-in-right': 'slideInRight 0.3s ease-out',
        'scale-in': 'scaleIn 0.2s ease-out',
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        shimmer: 'shimmer 2s linear infinite',
        glow: 'glow 2s ease-in-out infinite alternate'
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' }
        },
        slideUp: {
          '0%': { opacity: '0', transform: 'translateY(10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        },
        slideDown: {
          '0%': { opacity: '0', transform: 'translateY(-10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        },
        slideInRight: {
          '0%': { opacity: '0', transform: 'translateX(20px)' },
          '100%': { opacity: '1', transform: 'translateX(0)' }
        },
        scaleIn: {
          '0%': { opacity: '0', transform: 'scale(0.95)' },
          '100%': { opacity: '1', transform: 'scale(1)' }
        },
        shimmer: {
          '0%': { backgroundPosition: '-200% 0' },
          '100%': { backgroundPosition: '200% 0' }
        },
        glow: {
          '0%': { boxShadow: '0 0 20px rgba(124, 58, 237, 0.24)' },
          '100%': { boxShadow: '0 0 30px rgba(124, 58, 237, 0.38)' }
        }
      },
      backdropBlur: {
        xs: '2px'
      },
      borderRadius: {
        '4xl': '2rem',
        'workspace-card': 'var(--workspace-radius-card)',
        'workspace-button': 'var(--workspace-radius-button)',
        'workspace-input': 'var(--workspace-radius-input)'
      }
    }
  },
  plugins: []
}
