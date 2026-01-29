import { Link, useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { FiLogOut, FiUploadCloud, FiVideo, FiUser } from 'react-icons/fi';

const Navbar = () => {
    const { user, logout } = useAuth();
    const navigate = useNavigate();

    const handleLogout = () => {
        logout();
        navigate('/login');
    };

    return (
        <nav style={{ borderBottom: '1px solid var(--border-color)', background: 'rgba(15, 23, 42, 0.8)', backdropFilter: 'blur(10px)', position: 'sticky', top: 0, zIndex: 50 }}>
            <div className="container" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', height: '70px' }}>
                <Link to="/" style={{ fontSize: '1.5rem', fontWeight: 'bold', display: 'flex', alignItems: 'center', gap: '10px' }}>
                    <div style={{ width: '30px', height: '30px', background: 'linear-gradient(135deg, #6366f1, #a855f7)', borderRadius: '8px' }}></div>
                    <span style={{ background: 'linear-gradient(to right, #fff, #94a3b8)', WebkitBackgroundClip: 'text', WebkitTextFillColor: 'transparent' }}>g57</span>
                </Link>

                <div style={{ display: 'flex', gap: '20px', alignItems: 'center' }}>
                    {user ? (
                        <>
                            <Link to="/upload" className="btn btn-primary">
                                <FiUploadCloud style={{ marginRight: '8px', fontSize: '1.2em' }} /> Upload
                            </Link>
                            <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginLeft: '20px', borderLeft: '1px solid var(--border-color)', paddingLeft: '20px' }}>
                                <div style={{ textAlign: 'right', fontSize: '0.9rem' }}>
                                    <div style={{ fontWeight: '600' }}>{user.name}</div>
                                    <div style={{ color: 'var(--text-muted)', fontSize: '0.8rem' }}>{user.email}</div>
                                </div>
                                <button onClick={handleLogout} className="btn btn-outline" style={{ padding: '0.5rem', border: '1px solid var(--border-color)' }}>
                                    <FiLogOut />
                                </button>
                            </div>
                        </>
                    ) : (
                        <>
                            <Link to="/login" style={{ color: 'var(--text-muted)', marginRight: '1rem', fontWeight: '500' }}>Login</Link>
                            <Link to="/register" className="btn btn-primary">Começar Agora</Link>
                        </>
                    )}
                </div>
            </div>
        </nav>
    );
};

export default Navbar;
