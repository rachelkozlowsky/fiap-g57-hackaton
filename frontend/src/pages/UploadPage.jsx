import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import api from '../services/api';
import Navbar from '../components/Navbar';
import { FiUploadCloud, FiFile } from 'react-icons/fi';

const UploadPage = () => {
    const [file, setFile] = useState(null);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');
    const [progress, setProgress] = useState(0);
    const navigate = useNavigate();

    const handleFileChange = (e) => {
        if (e.target.files[0]) {
            setFile(e.target.files[0]);
            setError('');
        }
    };

    const handleSubmit = async (e) => {
        e.preventDefault();
        if (!file) {
            setError('Selecione um arquivo de vídeo');
            return;
        }

        const formData = new FormData();
        formData.append('video', file);

        setLoading(true);
        try {
            await api.post('/videos/upload', formData, {
                headers: {
                    'Content-Type': 'multipart/form-data',
                },
                onUploadProgress: (progressEvent) => {
                    const percentCompleted = Math.round((progressEvent.loaded * 100) / progressEvent.total);
                    setProgress(percentCompleted);
                },
            });
            navigate('/');
        } catch (err) {
            setError(err.response?.data?.message || 'Erro ao fazer upload do vídeo');
        } finally {
            setLoading(false);
        }
    };

    return (
        <>
            <Navbar />
            <div className="container" style={{ minHeight: 'calc(100vh - 70px)', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                <div className="card glass-panel" style={{ width: '100%', maxWidth: '500px' }}>
                    <h2 style={{ textAlign: 'center', marginBottom: '2rem' }}>Upload de Vídeo</h2>

                    {error && (
                        <div style={{ background: 'rgba(239, 68, 68, 0.2)', color: '#fca5a5', padding: '1rem', borderRadius: '0.5rem', marginBottom: '1rem', border: '1px solid rgba(239, 68, 68, 0.3)' }}>
                            {error}
                        </div>
                    )}

                    <form onSubmit={handleSubmit}>
                        <div
                            style={{
                                border: '2px dashed var(--border-color)',
                                borderRadius: '1rem',
                                padding: '3rem',
                                textAlign: 'center',
                                marginBottom: '2rem',
                                cursor: 'pointer',
                                background: file ? 'rgba(99, 102, 241, 0.1)' : 'transparent',
                                transition: 'all 0.2s'
                            }}
                            onClick={() => document.getElementById('video-input').click()}
                        >
                            <input
                                type="file"
                                id="video-input"
                                accept="video/*"
                                onChange={handleFileChange}
                                style={{ display: 'none' }}
                            />

                            {file ? (
                                <>
                                    <FiFile style={{ fontSize: '3rem', color: 'var(--primary-color)', marginBottom: '1rem' }} />
                                    <div style={{ fontWeight: '600' }}>{file.name}</div>
                                    <div style={{ color: 'var(--text-muted)', fontSize: '0.9rem' }}>{(file.size / 1024 / 1024).toFixed(2)} MB</div>
                                </>
                            ) : (
                                <>
                                    <FiUploadCloud style={{ fontSize: '3rem', color: 'var(--text-muted)', marginBottom: '1rem' }} />
                                    <div style={{ fontWeight: '600', marginBottom: '0.5rem' }}>Clique ou arraste para fazer upload</div>
                                    <div style={{ color: 'var(--text-muted)', fontSize: '0.9rem' }}>MP4, AVI, MKV (Max 500MB)</div>
                                </>
                            )}
                        </div>

                        {loading && (
                            <div style={{ marginBottom: '1.5rem' }}>
                                <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '0.5rem', fontSize: '0.9rem' }}>
                                    <span>Enviando...</span>
                                    <span>{progress}%</span>
                                </div>
                                <div style={{ width: '100%', height: '8px', background: 'var(--surface-color)', borderRadius: '4px', overflow: 'hidden' }}>
                                    <div style={{ width: `${progress}%`, height: '100%', background: 'var(--primary-color)', transition: 'width 0.3s' }}></div>
                                </div>
                            </div>
                        )}

                        <div style={{ display: 'flex', gap: '1rem' }}>
                            <button
                                type="button"
                                onClick={() => navigate('/')}
                                className="btn btn-outline"
                                style={{ flex: 1 }}
                                disabled={loading}
                            >
                                Cancelar
                            </button>
                            <button
                                type="submit"
                                className="btn btn-primary"
                                style={{ flex: 1 }}
                                disabled={loading || !file}
                            >
                                {loading ? 'Processando...' : 'Fazer Upload'}
                            </button>
                        </div>
                    </form>
                </div>
            </div>
        </>
    );
};

export default UploadPage;
