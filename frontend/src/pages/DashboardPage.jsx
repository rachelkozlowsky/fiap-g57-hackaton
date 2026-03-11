import { useState, useEffect } from 'react';
import api from '../services/api';
import Navbar from '../components/Navbar';
import ConfirmationModal from '../components/ConfirmationModal';
import { FiDownload, FiTrash2, FiClock, FiCheckCircle, FiAlertCircle, FiLoader } from 'react-icons/fi';

const DashboardPage = () => {
    const [videos, setVideos] = useState([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState('');
    const [deleteModal, setDeleteModal] = useState({ isOpen: false, videoId: null, isLoading: false });

    const fetchVideos = async () => {
        try {
            const response = await api.get('/videos');
            setVideos(response.data);
            setError('');
        } catch (err) {
            console.error(err);
            setError('Não foi possível carregar a lista de vídeos.');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchVideos();
        const interval = setInterval(fetchVideos, 5000);
        return () => clearInterval(interval);
    }, []);

    const handleDeleteClick = (id) => {
        setDeleteModal({ isOpen: true, videoId: id, isLoading: false });
    };

    const confirmDelete = async () => {
        if (!deleteModal.videoId) return;

        setDeleteModal(prev => ({ ...prev, isLoading: true }));
        try {
            await api.delete(`/videos/${deleteModal.videoId}`);
            setDeleteModal({ isOpen: false, videoId: null, isLoading: false });
            fetchVideos();
        } catch (err) {
            alert('Erro ao excluir vídeo');
            setDeleteModal(prev => ({ ...prev, isLoading: false }));
        }
    };

    const closeDeleteModal = () => {
        if (!deleteModal.isLoading) {
            setDeleteModal({ isOpen: false, videoId: null, isLoading: false });
        }
    };

    const getStatusBadge = (status) => {
        switch (status) {
            case 'completed':
                return <span style={{ color: 'var(--success-color)', display: 'flex', alignItems: 'center', gap: '5px' }}><FiCheckCircle /> Concluído</span>;
            case 'processing':
                return <span style={{ color: '#f59e0b', display: 'flex', alignItems: 'center', gap: '5px' }}><FiLoader className="spin" /> Processando</span>;
            case 'failed':
                return <span style={{ color: 'var(--error-color)', display: 'flex', alignItems: 'center', gap: '5px' }}><FiAlertCircle /> Falha</span>;
            default:
                return <span style={{ color: 'var(--text-muted)', display: 'flex', alignItems: 'center', gap: '5px' }}><FiClock /> Aguardando</span>;
        }
    };

    return (
        <>
            <Navbar />
            <div className="container" style={{ padding: '2rem 20px' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '2rem' }}>
                    <h1>Meus Vídeos</h1>
                    <button onClick={fetchVideos} className="btn btn-outline">Atualizar</button>
                </div>

                {error && <div style={{ color: 'var(--error-color)', marginBottom: '1rem' }}>{error}</div>}

                {loading && videos.length === 0 ? (
                    <div style={{ textAlign: 'center', padding: '4rem' }}>Carregando...</div>
                ) : videos.length === 0 ? (
                    <div style={{ textAlign: 'center', padding: '4rem', border: '1px dashed var(--border-color)', borderRadius: '1rem' }}>
                        <h3>Nenhum vídeo encontrado</h3>
                        <p style={{ color: 'var(--text-muted)', marginBottom: '1rem' }}>Faça o upload do seu primeiro vídeo para começar.</p>
                    </div>
                ) : (
                    <div className="grid-videos">
                        {videos.map((video) => (
                            <div key={video.id} className="card glass-panel animate-fade-in" style={{ padding: '1.5rem' }}>
                                <div style={{ marginBottom: '1rem' }}>
                                    <h3 style={{ fontSize: '1.2rem', marginBottom: '0.5rem', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }} title={video.original_name}>{video.original_name}</h3>
                                    <div style={{ fontSize: '0.8rem', color: 'var(--text-muted)' }}>
                                        {new Date(video.created_at).toLocaleString()}
                                    </div>
                                </div>

                                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1.5rem' }}>
                                    {getStatusBadge(video.status)}
                                </div>

                                <div style={{ display: 'flex', gap: '10px' }}>
                                    {video.status === 'completed' && (
                                        <a
                                            href={`http://localhost:8080/api/v1/videos/${video.id}/download`}
                                            target="_blank"
                                            rel="noopener noreferrer"
                                            className="btn btn-primary"
                                            style={{ flex: 1, fontSize: '0.9rem' }}
                                        >
                                            <FiDownload style={{ marginRight: '5px' }} /> Download
                                        </a>
                                    )}
                                    <button
                                        onClick={() => handleDeleteClick(video.id)}
                                        className="btn btn-outline"
                                        style={{ color: 'var(--error-color)', borderColor: 'rgba(239, 68, 68, 0.3)' }}
                                    >
                                        <FiTrash2 />
                                    </button>
                                </div>
                            </div>
                        ))}
                    </div>
                )}
            </div>

            <ConfirmationModal
                isOpen={deleteModal.isOpen}
                onClose={closeDeleteModal}
                onConfirm={confirmDelete}
                title="Excluir Vídeo"
                message="Tem certeza que deseja excluir este vídeo? Esta ação não pode ser desfeita."
                isLoading={deleteModal.isLoading}
            />
        </>
    );
};

export default DashboardPage;
